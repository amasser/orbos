package deployment

import (
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes/resources/deployment"
	"github.com/caos/orbos/internal/operator/zitadel"
	coredb "github.com/caos/orbos/internal/operator/zitadel/kinds/databases/core"
	corenw "github.com/caos/orbos/internal/operator/zitadel/kinds/networking/core"
	"github.com/caos/orbos/mntr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strings"
)

func AdaptFunc(
	monitor mntr.Monitor,
	namespace string,
	labels map[string]string,
	replicaCount int,
	version string,
	imagePullSecret string,
	cmName string,
	certPath string,
	secretName string,
	secretPath string,
	consoleCMName string,
	secretVarsName string,
	secretPasswordsName string,
	users []string,
	nodeSelector map[string]string,
	migrationDone zitadel.EnsureFunc,
	configurationDone zitadel.EnsureFunc,
	getConfigurationHashes func(currentDB coredb.DatabaseCurrent, currentNW corenw.NetworkingCurrent) map[string]string,
) (
	zitadel.QueryFunc,
	zitadel.DestroyFunc,
	zitadel.EnsureFunc,
	func(replicaCount int) zitadel.EnsureFunc,
	zitadel.EnsureFunc,
	error,
) {
	internalMonitor := monitor.WithField("component", "deployment")

	rootSecret := "client-root"
	secretMode := int32(0777)
	replicas := int32(replicaCount)
	runAsUser := int64(1000)
	runAsNonRoot := true
	certMountPath := "/dbsecrets"
	containerName := "zitadel"

	volumnes := []v1.Volume{{
		Name: secretName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretName,
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
		Name: secretPasswordsName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretPasswordsName,
			},
		},
	}, {
		Name: consoleCMName,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{Name: consoleCMName},
			},
		},
	}}
	volMounts := []v1.VolumeMount{
		{Name: secretName, MountPath: secretPath},
		{Name: consoleCMName, MountPath: "/console/environment.json", SubPath: "environment.json"},
		{Name: rootSecret, MountPath: certMountPath + "/ca.crt", SubPath: "ca.crt"},
	}

	for _, user := range users {
		userReplaced := strings.ReplaceAll(user, "_", "-")
		internalName := "client-" + userReplaced
		volumnes = append(volumnes, v1.Volume{
			Name: internalName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName:  "cockroachdb.client." + userReplaced,
					DefaultMode: &secretMode,
				},
			},
		})
		volMounts = append(volMounts, v1.VolumeMount{
			Name: internalName,
			//ReadOnly:  true,
			MountPath: certMountPath + "/client." + user + ".crt",
			SubPath:   "client." + user + ".crt",
		})
		volMounts = append(volMounts, v1.VolumeMount{
			Name: internalName,
			//ReadOnly:  true,
			MountPath: certMountPath + "/client." + user + ".key",
			SubPath:   "client." + user + ".key",
		})
	}

	envVars := []v1.EnvVar{
		{Name: "POD_IP",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			}},
		{Name: "CHAT_URL",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: secretVarsName},
					Key:                  "ZITADEL_GOOGLE_CHAT_URL",
				},
			}},
		{Name: "TWILIO_TOKEN",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: secretVarsName},
					Key:                  "ZITADEL_TWILIO_AUTH_TOKEN",
				},
			}},
		{Name: "TWILIO_SERVICE_SID",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: secretVarsName},
					Key:                  "ZITADEL_TWILIO_SID",
				},
			}},
		{Name: "SMTP_PASSWORD",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: secretVarsName},
					Key:                  "ZITADEL_EMAILAPPKEY",
				},
			}},
	}

	for _, user := range users {
		envVars = append(envVars, v1.EnvVar{
			Name: "CR_" + strings.ToUpper(user) + "_PASSWORD",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: secretPasswordsName},
					Key:                  user,
				},
			},
		})
	}

	deployName := "zitadel"
	maxUnavailable := intstr.FromInt(1)
	maxSurge := intstr.FromInt(1)

	deploymentDef := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deployName,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: map[string]string{},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: map[string]string{},
				},
				Spec: v1.PodSpec{
					NodeSelector: nodeSelector,
					SecurityContext: &v1.PodSecurityContext{
						RunAsUser:    &runAsUser,
						RunAsNonRoot: &runAsNonRoot,
					},
					Containers: []v1.Container{
						{
							Lifecycle: &v1.Lifecycle{
								PostStart: &v1.Handler{
									Exec: &v1.ExecAction{
										// TODO: until proper fix of https://github.com/kubernetes/kubernetes/issues/2630
										Command: []string{"sh", "-c",
											"mkdir -p " + certPath + "/ && cp " + certMountPath + "/* " + certPath + "/ && chmod 400 " + certPath + "/*"},
									},
								},
							},
							Args: []string{"start"},
							SecurityContext: &v1.SecurityContext{
								RunAsUser:    &runAsUser,
								RunAsNonRoot: &runAsNonRoot,
							},
							Name:            containerName,
							Image:           "docker.pkg.github.com/caos/zitadel/zitadel:" + version,
							ImagePullPolicy: "IfNotPresent",
							Ports: []v1.ContainerPort{
								{Name: "grpc", ContainerPort: 50001},
								{Name: "http", ContainerPort: 50002},
								{Name: "ui", ContainerPort: 50003},
							},
							Env: envVars,
							EnvFrom: []v1.EnvFromSource{
								{ConfigMapRef: &v1.ConfigMapEnvSource{
									LocalObjectReference: v1.LocalObjectReference{Name: cmName},
								}}},
							VolumeMounts: volMounts,
							LivenessProbe: &v1.Probe{
								Handler: v1.Handler{
									HTTPGet: &v1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.Parse("http"),
										Scheme: "HTTP",
									},
								},
								PeriodSeconds:    5,
								FailureThreshold: 2,
							},
							ReadinessProbe: &v1.Probe{
								Handler: v1.Handler{
									HTTPGet: &v1.HTTPGetAction{
										Path:   "/ready",
										Port:   intstr.Parse("http"),
										Scheme: "HTTP",
									},
								},
								PeriodSeconds:    5,
								FailureThreshold: 2,
							},
						},
					},
					ImagePullSecrets: []v1.LocalObjectReference{{
						Name: imagePullSecret,
					}},
					Volumes: volumnes,
				},
			},
		},
	}

	destroy, err := deployment.AdaptFuncToDestroy(namespace, deployName)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	destroyers := []zitadel.DestroyFunc{
		zitadel.ResourceDestroyToZitadelDestroy(destroy),
	}

	checkDeployRunning := func(k8sClient *kubernetes.Client) error {
		internalMonitor.Info("waiting for deployment to be running")
		if err := k8sClient.WaitUntilDeploymentReady(namespace, deployName, true, false, 60); err != nil {
			internalMonitor.Error(errors.Wrap(err, "error while waiting for deployment to be running"))
			return err
		}
		internalMonitor.Info("deployment is running")
		return nil
	}

	checkDeployNotReady := func(k8sClient *kubernetes.Client) error {
		internalMonitor.Info("checking for statefulset to not be ready")
		if err := k8sClient.WaitUntilStatefulsetIsReady(namespace, deployName, true, true, 1); err != nil {
			internalMonitor.Info("statefulset is not ready")
			return nil
		}
		internalMonitor.Info("statefulset is ready")
		return errors.New("statefulset is ready")
	}

	return func(k8sClient *kubernetes.Client, queried map[string]interface{}) (zitadel.EnsureFunc, error) {
			currentDB, err := coredb.ParseQueriedForDatabase(queried)
			if err != nil {
				return nil, err
			}

			currentNW, err := corenw.ParseQueriedForNetworking(queried)
			if err != nil {
				return nil, err
			}

			hashes := getConfigurationHashes(currentDB, currentNW)
			if hashes != nil && len(hashes) != 0 {
				for k, v := range hashes {
					deploymentDef.Annotations[k] = v
					deploymentDef.Spec.Template.Annotations[k] = v
				}
			}

			query, err := deployment.AdaptFuncToEnsure(deploymentDef)
			if err != nil {
				return nil, err
			}

			queriers := []zitadel.QueryFunc{
				zitadel.EnsureFuncToQueryFunc(currentDB.GetReadyQuery()),
				zitadel.EnsureFuncToQueryFunc(migrationDone),
				zitadel.EnsureFuncToQueryFunc(configurationDone),
				zitadel.ResourceQueryToZitadelQuery(query),
			}

			return zitadel.QueriersToEnsureFunc(internalMonitor, false, queriers, k8sClient, queried)
		},
		zitadel.DestroyersToDestroyFunc(internalMonitor, destroyers),
		func(k8sClient *kubernetes.Client) error {
			internalMonitor.Info("waiting for deployment to be ready")
			if err := k8sClient.WaitUntilDeploymentReady(namespace, deployName, true, true, 60); err != nil {
				internalMonitor.Error(errors.Wrap(err, "error while waiting for deployment to be ready"))
				return err
			}
			internalMonitor.Info("deployment is ready")
			return nil
		},
		func(replicaCount int) zitadel.EnsureFunc {
			return func(k8sClient *kubernetes.Client) error {
				internalMonitor.Info("Scaling deployment")
				return k8sClient.ScaleDeployment(namespace, deployName, replicaCount)
			}
		},
		func(k8sClient *kubernetes.Client) error {
			if err := checkDeployRunning(k8sClient); err != nil {
				return err
			}

			if err := checkDeployNotReady(k8sClient); err != nil {
				return nil
			}

			command := "/zitadel setup"

			if err := k8sClient.ExecInPodOfDeployment(namespace, deployName, containerName, command); err != nil {
				return err
			}
			return nil
		},
		nil
}
