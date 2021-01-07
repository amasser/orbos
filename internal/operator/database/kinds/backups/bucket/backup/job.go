package backup

import (
	"github.com/caos/orbos/internal/utils/helper"
	"github.com/caos/orbos/pkg/labels"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getCronJob(
	namespace string,
	nameLabels *labels.Name,
	cron string,
	jobSpecDef batchv1.JobSpec,
) *v1beta1.CronJob {
	return &v1beta1.CronJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      nameLabels.Name(),
			Namespace: namespace,
			Labels:    labels.MustK8sMap(nameLabels),
		},
		Spec: v1beta1.CronJobSpec{
			Schedule:          cron,
			ConcurrencyPolicy: v1beta1.ForbidConcurrent,
			JobTemplate: v1beta1.JobTemplateSpec{
				Spec: jobSpecDef,
			},
		},
	}
}

func getJob(
	namespace string,
	nameLabels *labels.Name,
	jobSpecDef batchv1.JobSpec,
) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: v1.ObjectMeta{
			Name:      nameLabels.Name(),
			Namespace: namespace,
			Labels:    labels.MustK8sMap(nameLabels),
		},
		Spec: jobSpecDef,
	}
}

func getJobSpecDef(
	nodeselector map[string]string,
	tolerations []corev1.Toleration,
	secretName string,
	secretKey string,
	backupName string,
	command string,
) batchv1.JobSpec {
	return batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				RestartPolicy: corev1.RestartPolicyNever,
				NodeSelector:  nodeselector,
				Tolerations:   tolerations,
				Containers: []corev1.Container{{
					Name:  backupName,
					Image: image,
					Command: []string{
						"/bin/bash",
						"-c",
						command,
					},
					VolumeMounts: []corev1.VolumeMount{{
						Name:      internalSecretName,
						MountPath: certPath,
					}, {
						Name:      secretKey,
						SubPath:   secretKey,
						MountPath: secretPath,
					}},
					ImagePullPolicy: corev1.PullAlways,
				}},
				Volumes: []corev1.Volume{{
					Name: internalSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  rootSecretName,
							DefaultMode: helper.PointerInt32(defaultMode),
						},
					},
				}, {
					Name: secretKey,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secretName,
						},
					},
				}},
			},
		},
	}
}
