package clean

import (
	"github.com/caos/orbos/internal/operator/core"
	"github.com/caos/orbos/mntr"
	"github.com/caos/orbos/pkg/kubernetes"
	"github.com/pkg/errors"
)

func getCleanupFunc(
	monitor mntr.Monitor,
	namespace string,
	jobName string,
) core.EnsureFunc {
	return func(k8sClient kubernetes.ClientInt) error {
		monitor.Info("waiting for clean to be completed")
		if err := k8sClient.WaitUntilJobCompleted(namespace, jobName, 60); err != nil {
			monitor.Error(errors.Wrap(err, "error while waiting for clean to be completed"))
			return err
		}
		monitor.Info("clean is completed, cleanup")
		if err := k8sClient.DeleteJob(namespace, jobName); err != nil {
			monitor.Error(errors.Wrap(err, "error while trying to cleanup clean"))
			return err
		}
		monitor.Info("clean cleanup is completed")
		return nil
	}
}
