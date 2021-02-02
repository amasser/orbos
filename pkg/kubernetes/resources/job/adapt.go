package job

import (
	"reflect"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/caos/orbos/pkg/kubernetes"
	"github.com/caos/orbos/pkg/kubernetes/resources"
	batch "k8s.io/api/batch/v1"
	macherrs "k8s.io/apimachinery/pkg/api/errors"
)

func AdaptFuncToEnsure(job *batch.Job) (resources.QueryFunc, error) {
	return func(k8sClient kubernetes.ClientInt) (resources.EnsureFunc, error) {

		jobDef, err := k8sClient.GetJob(job.GetNamespace(), job.GetName())
		if err != nil && !macherrs.IsNotFound(err) {
			return nil, err
		} else if macherrs.IsNotFound(err) {
			return func(k8sClient kubernetes.ClientInt) error {
				return k8sClient.ApplyJob(job)
			}, nil
		}

		//check if any immutable fields were changed
		changedImmutable := false
		if !reflect.DeepEqual(job.GetAnnotations(), jobDef.GetAnnotations()) {
			changedImmutable = true
		}
		if job.Spec.Selector != nil && !reflect.DeepEqual(job.Spec.Selector, jobDef.Spec.Selector) {
			changedImmutable = true
		}
		if job.Spec.Template.ObjectMeta.Labels != nil && !reflect.DeepEqual(job.Spec.Template.ObjectMeta.Labels, jobDef.Spec.Template.ObjectMeta.Labels) {
			changedImmutable = true
		}

		//workaround as securitycontext is a pointer to ensure that it only triggers if the values are different
		if job.Spec.Template.Spec.SecurityContext == nil {
			job.Spec.Template.Spec.SecurityContext = &v1.PodSecurityContext{}
		}

		if !reflect.DeepEqual(job.Spec.Template.Spec, jobDef.Spec.Template.Spec) {
			changedImmutable = true
		}

		if changedImmutable {
			return func(k8sClient kubernetes.ClientInt) error {
				if err := k8sClient.DeleteJob(job.GetNamespace(), job.GetName()); err != nil {
					return err
				}
				time.Sleep(1 * time.Second)
				return k8sClient.ApplyJob(job)
			}, nil
		}

		//check if selector or the labels are empty, as this have default values
		if job.Spec.Selector == nil {
			job.Spec.Selector = jobDef.Spec.Selector
		}
		if job.Spec.Template.ObjectMeta.Labels == nil {
			job.Spec.Template.ObjectMeta.Labels = jobDef.Spec.Template.ObjectMeta.Labels
		}

		return func(k8sClient kubernetes.ClientInt) error {
			return k8sClient.ApplyJob(job)
		}, nil

	}, nil
}

func AdaptFuncToDestroy(namespace, name string) (resources.DestroyFunc, error) {
	return func(client kubernetes.ClientInt) error {
		return client.DeleteJob(namespace, name)
	}, nil
}
