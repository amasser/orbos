package orb

import (
	"github.com/caos/orbos/internal/operator/core"
	"github.com/caos/orbos/internal/operator/networking/kinds/networking"
	"github.com/caos/orbos/mntr"
	kubernetes2 "github.com/caos/orbos/pkg/kubernetes"
	"github.com/caos/orbos/pkg/tree"
	"github.com/pkg/errors"
)

func AdaptFunc() core.AdaptFunc {

	namespaceStr := "caos-zitadel"
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "networking.caos.ch",
		"app.kubernetes.io/part-of":    "networking",
	}

	return func(monitor mntr.Monitor, desiredTree *tree.Tree, currentTree *tree.Tree) (queryFunc core.QueryFunc, destroyFunc core.DestroyFunc, err error) {
		defer func() {
			err = errors.Wrapf(err, "building %s failed", desiredTree.Common.Kind)
		}()

		orbMonitor := monitor.WithField("kind", "orb")

		desiredKind, err := ParseDesiredV0(desiredTree)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parsing desired state failed")
		}
		desiredTree.Parsed = desiredKind
		currentTree = &tree.Tree{}

		if desiredKind.Spec.Verbose && !orbMonitor.IsVerbose() {
			orbMonitor = orbMonitor.Verbose()
		}

		networkingCurrent := &tree.Tree{}
		queryNW, destroyNW, err := networking.GetQueryAndDestroyFuncs(orbMonitor, desiredKind.Networking, networkingCurrent, namespaceStr, labels)
		if err != nil {
			return nil, nil, err
		}

		queriers := []core.QueryFunc{
			core.EnsureFuncToQueryFunc(Reconcile(monitor, desiredTree)),
			queryNW,
		}

		destroyers := []core.DestroyFunc{
			destroyNW,
		}

		currentTree.Parsed = &DesiredV0{
			Common: &tree.Common{
				Kind:    "zitadel.caos.ch/Orb",
				Version: "v0",
			},
			Networking: networkingCurrent,
		}

		return func(k8sClient *kubernetes2.Client, _ map[string]interface{}) (core.EnsureFunc, error) {
				queried := map[string]interface{}{}
				monitor.WithField("queriers", len(queriers)).Info("Querying")
				return core.QueriersToEnsureFunc(monitor, true, queriers, k8sClient, queried)
			},
			func(k8sClient *kubernetes2.Client) error {
				monitor.WithField("destroyers", len(queriers)).Info("Destroy")
				return core.DestroyersToDestroyFunc(monitor, destroyers)(k8sClient)
			},
			nil
	}
}
