package deployment

import (
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes/resources"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes/resources/deployment"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type secret struct {
	Path      string
	Name      string
	Namespace string
}

func AdaptFunc(
	namespace string,
	labels map[string]string,
	replicaCount int,
	version string,
) (
	func(currentDB interface{}) (resources.EnsureFunc, error),
	resources.DestroyFunc,
	error,
) {
	internalSecrets := "zitadel-secret"
	internalConfig := "console-config"
	rootSecret := "client-root"
	secretMode := int32(0400)
	replicas := int32(replicaCount)
	runAsUser := int64(1000)
	runAsNonRoot := true

	userList := []string{"management", "auth", "authz", "admin", "notify"}
	volumnes := []v1.Volume{{
		Name: internalSecrets,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: "zitadel-secret",
			},
		},
	}, {
		Name: rootSecret,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName:  "cockroachdb.client.root",
				DefaultMode: &secretMode,
			},
		},
	}, {
		Name: internalConfig,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{Name: "console-config"},
			},
		},
	}}
	volMounts := []v1.VolumeMount{
		{Name: internalSecrets, MountPath: "/secret"},
		{Name: internalConfig, MountPath: "/console/environment.json", SubPath: "environment.json"},
		{Name: rootSecret, MountPath: "/dbsecrets/ca.crt", SubPath: "ca.crt"},
	}
	for _, user := range userList {
		internalName := "client-" + user
		volumnes = append(volumnes, v1.Volume{
			Name: internalName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName:  "cockroachdb.client." + user,
					DefaultMode: &secretMode,
				},
			},
		})
		volMounts = append(volMounts, v1.VolumeMount{
			Name: internalName,
			//ReadOnly:  true,
			MountPath: "/dbsecrets/client." + user + ".crt",
			SubPath:   "client." + user + ".crt",
		})
		volMounts = append(volMounts, v1.VolumeMount{
			Name: internalName,
			//ReadOnly:  true,
			MountPath: "/dbsecrets/client." + user + ".key",
			SubPath:   "client." + user + ".key",
		})
	}
	deploymentDef := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "zitadel",
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},

				Spec: v1.PodSpec{
					SecurityContext: &v1.PodSecurityContext{
						RunAsUser:    &runAsUser,
						RunAsNonRoot: &runAsNonRoot,
					},
					Containers: []v1.Container{
						{
							SecurityContext: &v1.SecurityContext{
								RunAsUser:    &runAsUser,
								RunAsNonRoot: &runAsNonRoot,
							},
							Command:         []string{"/bin/sh", "-c", "mkdir -p /dbsecrets-zitadel/ && cp /dbsecrets/* /dbsecrets-zitadel/ && chmod 400 /dbsecrets-zitadel/* && chown 1000:1000 /dbsecrets-zitadel/* && while true; do sleep 30; done;"},
							Name:            "zitadel",
							Image:           "docker.pkg.github.com/caos/zitadel/zitadel:" + version,
							ImagePullPolicy: "IfNotPresent",
							Ports: []v1.ContainerPort{
								{Name: "management-rest", ContainerPort: 50011},
								{Name: "management-grpc", ContainerPort: 50010},
								{Name: "auth-rest", ContainerPort: 50021},
								{Name: "issuer-rest", ContainerPort: 50022},
								{Name: "auth-grpc", ContainerPort: 50020},
								{Name: "admin-rest", ContainerPort: 50041},
								{Name: "admin-grpc", ContainerPort: 50040},
								{Name: "console-http", ContainerPort: 50050},
								{Name: "accounts-http", ContainerPort: 50031},
							},
							Env: []v1.EnvVar{
								{Name: "POD_IP",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									}},
								{Name: "CHAT_URL",
									ValueFrom: &v1.EnvVarSource{
										SecretKeyRef: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{Name: "zitadel-secrets-vars"},
											Key:                  "ZITADEL_GOOGLE_CHAT_URL",
										},
									}},
								{Name: "TWILIO_TOKEN",
									ValueFrom: &v1.EnvVarSource{
										SecretKeyRef: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{Name: "zitadel-secrets-vars"},
											Key:                  "ZITADEL_TWILIO_AUTH_TOKEN",
										},
									}},
								{Name: "TWILIO_SERVICE_SID",
									ValueFrom: &v1.EnvVarSource{
										SecretKeyRef: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{Name: "zitadel-secrets-vars"},
											Key:                  "ZITADEL_TWILIO_SID",
										},
									}},
								{Name: "SMTP_PASSWORD",
									ValueFrom: &v1.EnvVarSource{
										SecretKeyRef: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{Name: "zitadel-secrets-vars"},
											Key:                  "ZITADEL_EMAILAPPKEY",
										},
									}},
							},
							EnvFrom: []v1.EnvFromSource{
								{ConfigMapRef: &v1.ConfigMapEnvSource{
									LocalObjectReference: v1.LocalObjectReference{Name: "zitadel-vars"},
								}}},
							VolumeMounts: volMounts,
						},
					},
					ImagePullSecrets: []v1.LocalObjectReference{{
						Name: "public-github-packages",
					}},
					Volumes: volumnes,
				},
			},
		},
	}

	_, destroy, err := deployment.AdaptFunc(deploymentDef)
	if err != nil {
		return nil, nil, err
	}

	return func(currentDB interface{}) (resources.EnsureFunc, error) {
		query, _, err := deployment.AdaptFunc(deploymentDef)
		if err != nil {
			return nil, err
		}
		return query()
	}, destroy, nil
}
