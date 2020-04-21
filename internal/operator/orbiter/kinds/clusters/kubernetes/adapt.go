package kubernetes

import (
	"os"

	core "k8s.io/api/core/v1"

	"github.com/pkg/errors"

	"github.com/caos/orbiter/internal/operator/common"
	"github.com/caos/orbiter/internal/operator/orbiter"
	"github.com/caos/orbiter/mntr"
)

var deployErrors int

func AdaptFunc(
	orb *orbiter.Orb,
	orbiterCommit string,
	id string,
	oneoff bool,
	deployOrbiterAndBoom bool,
	destroyProviders func() (map[string]interface{}, error),
	whitelist func([]*orbiter.CIDR)) orbiter.AdaptFunc {

	return func(monitor mntr.Monitor, desiredTree *orbiter.Tree, currentTree *orbiter.Tree) (queryFunc orbiter.QueryFunc, destroyFunc orbiter.DestroyFunc, secrets map[string]*orbiter.Secret, migrate bool, err error) {
		defer func() {
			err = errors.Wrapf(err, "building %s failed", desiredTree.Common.Kind)
		}()

		if desiredTree.Common.Version != "v0" {
			migrate = true
		}

		desiredKind := &DesiredV0{
			Common: *desiredTree.Common,
			Spec:   Spec{Kubeconfig: &orbiter.Secret{Masterkey: orb.Masterkey}},
		}
		if err := desiredTree.Original.Decode(desiredKind); err != nil {
			return nil, nil, nil, migrate, errors.Wrap(err, "parsing desired state failed")
		}
		desiredTree.Parsed = desiredKind

		if desiredKind.Spec.ControlPlane.Taints == nil {
			taints := Taints([]Taint{{
				Key:    "node-role.kubernetes.io/master",
				Effect: core.TaintEffectNoSchedule,
			}})
			desiredKind.Spec.ControlPlane.Taints = &taints
			migrate = true
		}

		if err := desiredKind.validate(); err != nil {
			return nil, nil, nil, migrate, err
		}

		if desiredKind.Spec.Kubeconfig == nil {
			desiredKind.Spec.Kubeconfig = &orbiter.Secret{Masterkey: orb.Masterkey}
		}

		if desiredKind.Spec.Verbose && !monitor.IsVerbose() {
			monitor = monitor.Verbose()
		}

		whitelist([]*orbiter.CIDR{&desiredKind.Spec.Networking.PodCidr})

		var kc *string
		if desiredKind.Spec.Kubeconfig.Value != "" {
			kc = &desiredKind.Spec.Kubeconfig.Value
		}
		k8sClient := NewK8sClient(monitor, kc)

		if k8sClient.Available() && deployOrbiterAndBoom {
			if err := ensureArtifacts(monitor, k8sClient, orb, desiredKind.Spec.Versions.Orbiter, desiredKind.Spec.Versions.Boom); err != nil {
				deployErrors++
				monitor.WithFields(map[string]interface{}{
					"count": deployErrors,
					"error": err.Error(),
				}).Info("Deploying Orbiter failed, awaiting next iteration")
				if deployErrors > 50 {
					panic(err)
				}
			} else {
				if oneoff {
					monitor.Info("Deployed Orbiter takes over control")
					os.Exit(0)
				}
				deployErrors = 0
			}
		}

		current := &CurrentCluster{}
		currentTree.Parsed = &Current{
			Common: orbiter.Common{
				Kind:    "orbiter.caos.ch/KubernetesCluster",
				Version: "v0",
			},
			Current: current,
		}

		return func(nodeAgentsCurrent map[string]*common.NodeAgentCurrent, nodeAgentsDesired map[string]*common.NodeAgentSpec, providers map[string]interface{}) (orbiter.EnsureFunc, error) {
				ensureFunc, err := query(monitor,
					desiredKind,
					current,
					providers,
					nodeAgentsCurrent,
					nodeAgentsDesired,
					k8sClient,
					orb.URL,
					orb.Repokey,
					orbiterCommit,
					oneoff)
				return ensureFunc, errors.Wrapf(err, "querying %s failed", desiredKind.Common.Kind)
			}, func() error {
				defer func() {
					err = errors.Wrapf(err, "destroying %s failed", desiredKind.Common.Kind)
				}()

				providers, err := destroyProviders()
				if err != nil {
					return err
				}

				desiredKind.Spec.Kubeconfig = nil

				return destroy(monitor, providers, k8sClient)
			}, map[string]*orbiter.Secret{
				"kubeconfig": desiredKind.Spec.Kubeconfig,
			}, migrate, nil
	}
}
