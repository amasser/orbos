package job

import (
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes/resources"
	batch "k8s.io/api/batch/v1"
	macherrs "k8s.io/apimachinery/pkg/api/errors"
	"reflect"
	"time"
)

func AdaptFuncToEnsure(job *batch.Job) (resources.QueryFunc, error) {
	return func(k8sClient *kubernetes.Client) (resources.EnsureFunc, error) {

		jobDef, err := k8sClient.GetJob(job.GetNamespace(), job.GetName())
		if err != nil && !macherrs.IsNotFound(err) {
			return nil, err
		} else if macherrs.IsNotFound(err) {
			return func(k8sClient *kubernetes.Client) error {
				return k8sClient.ApplyJob(job)
			}, nil
		}

		changedImmutable := false
		if !reflect.DeepEqual(job.GetAnnotations(), jobDef.GetAnnotations()) {
			changedImmutable = true
		}

		if !reflect.DeepEqual(job.Spec.Template.Spec, jobDef.Spec.Template.Spec) &&
			//workaround as securitycontext is a pointer to ensure that it only triggers if the values are different
			!reflect.DeepEqual(*job.Spec.Template.Spec.SecurityContext, *jobDef.Spec.Template.Spec.SecurityContext) {
			changedImmutable = true
		}

		if changedImmutable {
			return func(k8sClient *kubernetes.Client) error {
				if err := k8sClient.DeleteJob(job.GetNamespace(), job.GetName()); err != nil {
					return err
				}
				time.Sleep(1 * time.Second)
				return k8sClient.ApplyJob(job)
			}, nil
		}

		return func(k8sClient *kubernetes.Client) error {
			return nil
		}, nil

	}, nil
}

func AdaptFuncToDestroy(namespace, name string) (resources.DestroyFunc, error) {
	return func(client *kubernetes.Client) error {
		return client.DeleteJob(namespace, name)
	}, nil
}
