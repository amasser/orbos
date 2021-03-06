package orb

import (
	"github.com/caos/orbos/internal/operator/core"
	"github.com/caos/orbos/mntr"
	"github.com/caos/orbos/pkg/kubernetes"
	"github.com/caos/orbos/pkg/tree"
	"github.com/caos/orbos/pkg/treelabels"
	"github.com/pkg/errors"
)

func Reconcile(monitor mntr.Monitor, desiredTree *tree.Tree) core.EnsureFunc {
	return func(k8sClient kubernetes.ClientInt) (err error) {
		defer func() {
			err = errors.Wrapf(err, "building %s failed", desiredTree.Common.Kind)
		}()

		desiredKind, err := ParseDesiredV0(desiredTree)
		if err != nil {
			return errors.Wrap(err, "parsing desired state failed")
		}
		desiredTree.Parsed = desiredKind

		recMonitor := monitor.WithField("version", desiredKind.Spec.Version)

		if desiredKind.Spec.Version == "" {
			err := errors.New("No version set in networking.yml")
			monitor.Error(err)
			return err
		}

		imageRegistry := desiredKind.Spec.CustomImageRegistry
		if imageRegistry == "" {
			imageRegistry = "ghcr.io"
		}
		if err := kubernetes.EnsureNetworkingArtifacts(monitor, treelabels.MustForAPI(desiredTree, mustDatabaseOperator(&desiredKind.Spec.Version)), k8sClient, desiredKind.Spec.Version, desiredKind.Spec.NodeSelector, desiredKind.Spec.Tolerations, imageRegistry); err != nil {
			recMonitor.Error(errors.Wrap(err, "Failed to deploy networking-operator into k8s-cluster"))
			return err
		}

		recMonitor.Info("Applied networking-operator")

		return nil

	}
}
