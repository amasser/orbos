package gce

import (
	"github.com/caos/orbos/internal/operator/orbiter/kinds/loadbalancers"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/loadbalancers/dynamic"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/providers/core"
	"github.com/caos/orbos/internal/orb"
	"github.com/caos/orbos/internal/secret"
	"github.com/caos/orbos/internal/ssh"
	"github.com/caos/orbos/internal/tree"
	"github.com/pkg/errors"

	"github.com/caos/orbos/internal/operator/common"
	"github.com/caos/orbos/internal/operator/orbiter"
	"github.com/caos/orbos/mntr"
)

func AdaptFunc(providerID, orbID string, whitelist dynamic.WhiteListFunc, orbiterCommit, repoURL, repoKey string, oneoff bool) orbiter.AdaptFunc {
	return func(monitor mntr.Monitor, finishedChan chan struct{}, desiredTree *tree.Tree, currentTree *tree.Tree) (queryFunc orbiter.QueryFunc, destroyFunc orbiter.DestroyFunc, configureFunc orbiter.ConfigureFunc, migrate bool, err error) {
		defer func() {
			err = errors.Wrapf(err, "building %s failed", desiredTree.Common.Kind)
		}()
		desiredKind, err := parseDesiredV0(desiredTree)
		if err != nil {
			return nil, nil, nil, migrate, errors.Wrap(err, "parsing desired state failed")
		}
		desiredTree.Parsed = desiredKind

		if desiredKind.Spec.RebootRequired == nil {
			desiredKind.Spec.RebootRequired = make([]string, 0)
			migrate = true
		}

		if desiredKind.Spec.Verbose && !monitor.IsVerbose() {
			monitor = monitor.Verbose()
		}

		if err := desiredKind.validate(); err != nil {
			return nil, nil, nil, migrate, err
		}

		lbCurrent := &tree.Tree{}
		var lbQuery orbiter.QueryFunc

		lbQuery, lbDestroy, lbConfigure, migrateLocal, err := loadbalancers.GetQueryAndDestroyFunc(monitor, whitelist, desiredKind.Loadbalancing, lbCurrent, finishedChan)
		if err != nil {
			return nil, nil, nil, migrate, err
		}
		if migrateLocal {
			migrate = true
		}

		current := &Current{
			Common: &tree.Common{
				Kind:    "orbiter.caos.ch/GCEProvider",
				Version: "v0",
			},
		}
		currentTree.Parsed = current

		return func(nodeAgentsCurrent *common.CurrentNodeAgents, nodeAgentsDesired *common.DesiredNodeAgents, _ map[string]interface{}) (ensureFunc orbiter.EnsureFunc, err error) {
				defer func() {
					err = errors.Wrapf(err, "querying %s failed", desiredKind.Common.Kind)
				}()

				if _, err := lbQuery(nodeAgentsCurrent, nodeAgentsDesired, nil); err != nil {
					return nil, err
				}

				_, naFuncs := core.NodeAgentFuncs(monitor, repoURL, repoKey)
				ctx, err := buildContext(monitor, &desiredKind.Spec, orbID, providerID, oneoff)
				if err != nil {
					return nil, err
				}

				return query(&desiredKind.Spec, current, lbCurrent.Parsed, ctx, nodeAgentsCurrent, nodeAgentsDesired, naFuncs, orbiterCommit)
			}, func() error {
				if err := lbDestroy(); err != nil {
					return err
				}
				ctx, err := buildContext(monitor, &desiredKind.Spec, orbID, providerID, oneoff)
				if err != nil {
					return err
				}

				return destroy(ctx)
			}, func(orb orb.Orb) error {
				if err := lbConfigure(orb); err != nil {
					return err
				}

				if desiredKind.Spec.SSHKey == nil ||
					desiredKind.Spec.SSHKey.Private == nil || desiredKind.Spec.SSHKey.Private.Value == "" ||
					desiredKind.Spec.SSHKey.Public == nil || desiredKind.Spec.SSHKey.Public.Value == "" {
					priv, pub, err := ssh.Generate()
					if err != nil {
						return err
					}
					desiredKind.Spec.SSHKey = &SSHKey{
						Private: &secret.Secret{Value: priv},
						Public:  &secret.Secret{Value: pub},
					}
				}

				if desiredKind.Spec.JSONKey == nil {
					// TODO: Create service account and write its json key to desiredKind.Spec.JSONKey and push repo
					return nil
				}

				ctx, err := buildContext(monitor, &desiredKind.Spec, orbID, providerID, true)
				if err != nil {
					return err
				}

				return core.ConfigureNodeAgents(ctx.machinesService, ctx.monitor, orb)
			}, migrate, nil
	}
}
