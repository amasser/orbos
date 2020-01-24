package kubernetes

import (
	"github.com/pkg/errors"

	"github.com/caos/orbiter/internal/operator/common"
	"github.com/caos/orbiter/internal/operator/orbiter"
	"github.com/caos/orbiter/logging"
)

func AdaptFunc(
	logger logging.Logger,
	orb *orbiter.Orb,
	orbiterCommit string,
	id string,
	oneoff bool,
	deployOrbiterAndBoom bool,
	ensureProviders func(psf orbiter.PushSecretsFunc, nodeAgentsCurrent map[string]*common.NodeAgentCurrent, nodeAgentsDesired map[string]*common.NodeAgentSpec) (map[string]interface{}, error),
	destroyProviders func() (map[string]interface{}, error)) orbiter.AdaptFunc {

	var deployErrors int
	return func(desiredTree *orbiter.Tree, secretsTree *orbiter.Tree, currentTree *orbiter.Tree) (ensureFunc orbiter.EnsureFunc, destroyFunc orbiter.DestroyFunc, secrets map[string]*orbiter.Secret, err error) {
		defer func() {
			err = errors.Wrapf(err, "building %s failed", desiredTree.Common.Kind)
		}()

		desiredKind := &DesiredV0{Common: *desiredTree.Common}
		if err := desiredTree.Original.Decode(desiredKind); err != nil {
			return nil, nil, nil, errors.Wrap(err, "parsing desired state failed")
		}
		desiredKind.Common.Version = "v0"
		desiredTree.Parsed = desiredKind

		if err := desiredKind.validate(); err != nil {
			return nil, nil, nil, err
		}

		if desiredKind.Spec.Verbose && !logger.IsVerbose() {
			logger = logger.Verbose()
		}

		secretsKind := &SecretsV0{
			Common:  *secretsTree.Common,
			Secrets: Secrets{Kubeconfig: &orbiter.Secret{Masterkey: orb.Masterkey}},
		}
		if err := secretsTree.Original.Decode(secretsKind); err != nil {
			return nil, nil, nil, errors.Wrap(err, "parsing secrets failed")
		}
		secretsKind.Common.Version = "v0"
		secretsTree.Parsed = secretsKind

		if secretsKind.Secrets.Kubeconfig == nil {
			secretsKind.Secrets.Kubeconfig = &orbiter.Secret{Masterkey: orb.Masterkey}
		}

		if deployOrbiterAndBoom && secretsKind.Secrets.Kubeconfig.Value != "" {
			if err := ensureArtifacts(logger, secretsKind.Secrets.Kubeconfig, orb, oneoff, desiredKind.Spec.Versions.Orbiter, desiredKind.Spec.Versions.Boom); err != nil {
				deployErrors++
				logger.WithFields(map[string]interface{}{
					"count": deployErrors,
					"msg":   "Deploying Orbiter failed, awaiting next iteration",
				}).Error(err)
				if deployErrors > 50 {
					panic(err)
				}
			} else {
				deployErrors = 0
			}
		}

		current := &CurrentCluster{}
		currentTree.Parsed = &Current{
			Common: orbiter.Common{
				Kind:    "orbiter.caos.ch/KubernetesCluster",
				Version: "v0",
			},
			Current: *current,
		}

		return func(psf orbiter.PushSecretsFunc, nodeAgentsCurrent map[string]*common.NodeAgentCurrent, nodeAgentsDesired map[string]*common.NodeAgentSpec) (err error) {
				defer func() {
					err = errors.Wrapf(err, "ensuring %s failed", desiredKind.Common.Kind)
				}()

				providers, err := ensureProviders(psf, nodeAgentsCurrent, nodeAgentsDesired)
				if err != nil {
					return err
				}

				return ensure(
					logger,
					*desiredKind,
					current,
					providers,
					nodeAgentsCurrent,
					nodeAgentsDesired,
					psf,
					secretsKind.Secrets.Kubeconfig,
					orb.URL,
					orb.Repokey,
					orbiterCommit,
					oneoff)
			}, func() error {
				defer func() {
					err = errors.Wrapf(err, "destroying %s failed", desiredKind.Common.Kind)
				}()

				providers, err := destroyProviders()
				if err != nil {
					return err
				}

				return destroy(providers, secretsKind.Secrets.Kubeconfig)
			}, map[string]*orbiter.Secret{
				"kubeconfig": secretsKind.Secrets.Kubeconfig,
			}, nil
	}
}
