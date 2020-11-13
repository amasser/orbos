package kubernetes

import (
	"fmt"

	"github.com/caos/orbos/internal/orb"
	"github.com/caos/orbos/pkg/kubernetes/k8s"

	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/api/resource"

	"gopkg.in/yaml.v2"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	mach "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/caos/orbos/mntr"
)

func EnsureCommonArtifacts(monitor mntr.Monitor, client *Client) error {

	monitor.Debug("Ensuring common artifacts")

	return client.ApplyNamespace(&core.Namespace{
		ObjectMeta: mach.ObjectMeta{
			Name: "caos-system",
			Labels: map[string]string{
				"name": "caos-system",
			},
		},
	})
}

func EnsureConfigArtifacts(monitor mntr.Monitor, client *Client, orb *orb.Orb) error {
	monitor.Debug("Ensuring configuration artifacts")

	orbfile, err := yaml.Marshal(orb)
	if err != nil {
		return err
	}

	if err := client.ApplySecret(&core.Secret{
		ObjectMeta: mach.ObjectMeta{
			Name:      "caos",
			Namespace: "caos-system",
		},
		StringData: map[string]string{
			"orbconfig": string(orbfile),
		},
	}); err != nil {
		return err
	}

	return nil
}

func EnsureDatabaseArtifacts(
	monitor mntr.Monitor,
	client *Client,
	version string,
	nodeselector map[string]string,
	tolerations []core.Toleration,
	imageRegistry string) error {

	monitor.WithFields(map[string]interface{}{
		"database": version,
	}).Debug("Ensuring database artifacts")

	if version == "" {
		return nil
	}

	if err := client.ApplyServiceAccount(&core.ServiceAccount{
		ObjectMeta: mach.ObjectMeta{
			Name:      "database-operator",
			Namespace: "caos-system",
		},
	}); err != nil {
		return err
	}

	if err := client.ApplyClusterRole(&rbac.ClusterRole{
		ObjectMeta: mach.ObjectMeta{
			Name: "database-operator-clusterrole",
			Labels: map[string]string{
				"app.kubernetes.io/instance":  "database",
				"app.kubernetes.io/part-of":   "orbos",
				"app.kubernetes.io/component": "database",
			},
		},
		Rules: []rbac.PolicyRule{{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		}},
	}); err != nil {
		return err
	}

	if err := client.ApplyClusterRoleBinding(&rbac.ClusterRoleBinding{
		ObjectMeta: mach.ObjectMeta{
			Name: "database-operator-clusterrolebinding",
			Labels: map[string]string{
				"app.kubernetes.io/instance":  "database",
				"app.kubernetes.io/part-of":   "orbos",
				"app.kubernetes.io/component": "database",
			},
		},

		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "database-operator-clusterrole",
		},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      "database-operator",
			Namespace: "caos-system",
		}},
	}); err != nil {
		return err
	}

	if err := client.ApplyDeployment(&apps.Deployment{
		ObjectMeta: mach.ObjectMeta{
			Name:      "database-operator",
			Namespace: "caos-system",
			Labels: map[string]string{
				"app.kubernetes.io/instance":   "database",
				"app.kubernetes.io/part-of":    "orbos",
				"app.kubernetes.io/component":  "database",
				"app.kubernetes.io/managed-by": "database.caos.ch",
			},
		},
		Spec: apps.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &mach.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance":  "database",
					"app.kubernetes.io/part-of":   "orbos",
					"app.kubernetes.io/component": "database",
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: mach.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/instance":  "database",
						"app.kubernetes.io/part-of":   "orbos",
						"app.kubernetes.io/component": "database",
					},
				},
				Spec: core.PodSpec{
					ServiceAccountName: "database-operator",
					Containers: []core.Container{{
						Name:            "database",
						ImagePullPolicy: core.PullIfNotPresent,
						Image:           fmt.Sprintf("%s/caos/orbos:%s", imageRegistry, version),
						Command:         []string{"/orbctl", "takeoff", "database", "-f", "/secrets/orbconfig"},
						Args:            []string{},
						Ports: []core.ContainerPort{{
							Name:          "metrics",
							ContainerPort: 2112,
							Protocol:      "TCP",
						}},
						VolumeMounts: []core.VolumeMount{{
							Name:      "orbconfig",
							ReadOnly:  true,
							MountPath: "/secrets",
						}},
						Resources: core.ResourceRequirements{
							Limits: core.ResourceList{
								"cpu":    resource.MustParse("500m"),
								"memory": resource.MustParse("500Mi"),
							},
							Requests: core.ResourceList{
								"cpu":    resource.MustParse("250m"),
								"memory": resource.MustParse("250Mi"),
							},
						},
					}},
					NodeSelector: nodeselector,
					Tolerations:  tolerations,
					Volumes: []core.Volume{{
						Name: "orbconfig",
						VolumeSource: core.VolumeSource{
							Secret: &core.SecretVolumeSource{
								SecretName: "caos",
							},
						},
					}},
					TerminationGracePeriodSeconds: int64Ptr(10),
				},
			},
		},
	}); err != nil {
		return err
	}
	monitor.WithFields(map[string]interface{}{
		"version": version,
	}).Debug("Database Operator deployment ensured")

	return nil
}

func EnsureNetworkingArtifacts(
	monitor mntr.Monitor,
	client *Client,
	version string,
	nodeselector map[string]string,
	tolerations []core.Toleration,
	imageRegistry string) error {

	monitor.WithFields(map[string]interface{}{
		"networking": version,
	}).Debug("Ensuring networking artifacts")

	if version == "" {
		return nil
	}

	if err := client.ApplyServiceAccount(&core.ServiceAccount{
		ObjectMeta: mach.ObjectMeta{
			Name:      "networking-operator",
			Namespace: "caos-system",
		},
	}); err != nil {
		return err
	}

	if err := client.ApplyClusterRole(&rbac.ClusterRole{
		ObjectMeta: mach.ObjectMeta{
			Name: "networking-operator-clusterrole",
			Labels: map[string]string{
				"app.kubernetes.io/instance":  "networking",
				"app.kubernetes.io/part-of":   "orbos",
				"app.kubernetes.io/component": "networking",
			},
		},
		Rules: []rbac.PolicyRule{{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		}},
	}); err != nil {
		return err
	}

	if err := client.ApplyClusterRoleBinding(&rbac.ClusterRoleBinding{
		ObjectMeta: mach.ObjectMeta{
			Name: "networking-operator-clusterrolebinding",
			Labels: map[string]string{
				"app.kubernetes.io/instance":  "networking",
				"app.kubernetes.io/part-of":   "orbos",
				"app.kubernetes.io/component": "networking",
			},
		},

		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "networking-operator-clusterrole",
		},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      "networking-operator",
			Namespace: "caos-system",
		}},
	}); err != nil {
		return err
	}

	if err := client.ApplyDeployment(&apps.Deployment{
		ObjectMeta: mach.ObjectMeta{
			Name:      "networking-operator",
			Namespace: "caos-system",
			Labels: map[string]string{
				"app.kubernetes.io/instance":   "networking",
				"app.kubernetes.io/part-of":    "orbos",
				"app.kubernetes.io/component":  "networking",
				"app.kubernetes.io/managed-by": "networking.caos.ch",
			},
		},
		Spec: apps.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &mach.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance":  "networking",
					"app.kubernetes.io/part-of":   "orbos",
					"app.kubernetes.io/component": "networking",
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: mach.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/instance":  "networking",
						"app.kubernetes.io/part-of":   "orbos",
						"app.kubernetes.io/component": "networking",
					},
				},
				Spec: core.PodSpec{
					ServiceAccountName: "networking-operator",
					Containers: []core.Container{{
						Name:            "networking",
						ImagePullPolicy: core.PullIfNotPresent,
						Image:           fmt.Sprintf("%s/caos/orbos:%s", imageRegistry, version),
						Command:         []string{"/orbctl", "takeoff", "networking", "-f", "/secrets/orbconfig"},
						Args:            []string{},
						Ports: []core.ContainerPort{{
							Name:          "metrics",
							ContainerPort: 2112,
							Protocol:      "TCP",
						}},
						VolumeMounts: []core.VolumeMount{{
							Name:      "orbconfig",
							ReadOnly:  true,
							MountPath: "/secrets",
						}},
						Resources: core.ResourceRequirements{
							Limits: core.ResourceList{
								"cpu":    resource.MustParse("500m"),
								"memory": resource.MustParse("500Mi"),
							},
							Requests: core.ResourceList{
								"cpu":    resource.MustParse("250m"),
								"memory": resource.MustParse("250Mi"),
							},
						},
					}},
					NodeSelector: nodeselector,
					Tolerations:  tolerations,
					Volumes: []core.Volume{{
						Name: "orbconfig",
						VolumeSource: core.VolumeSource{
							Secret: &core.SecretVolumeSource{
								SecretName: "caos",
							},
						},
					}},
					TerminationGracePeriodSeconds: int64Ptr(10),
				},
			},
		},
	}); err != nil {
		return err
	}
	monitor.WithFields(map[string]interface{}{
		"version": version,
	}).Debug("Networking Operator deployment ensured")

	return nil
}

func EnsureBoomArtifacts(
	monitor mntr.Monitor,
	client *Client,
	version string,
	tolerations k8s.Tolerations,
	nodeselector map[string]string,
	resources *k8s.Resources,
	imageRegistry string) error {

	monitor.WithFields(map[string]interface{}{
		"boom": version,
	}).Debug("Ensuring boom artifacts")

	if version == "" {
		return nil
	}

	if err := client.ApplyServiceAccount(&core.ServiceAccount{
		ObjectMeta: mach.ObjectMeta{
			Name:      "boom",
			Namespace: "caos-system",
		},
	}); err != nil {
		return err
	}

	if err := client.ApplyClusterRole(&rbac.ClusterRole{
		ObjectMeta: mach.ObjectMeta{
			Name: "boom-clusterrole",
			Labels: map[string]string{
				"app.kubernetes.io/instance":  "boom",
				"app.kubernetes.io/part-of":   "orbos",
				"app.kubernetes.io/component": "boom",
			},
		},
		Rules: []rbac.PolicyRule{{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		}},
	}); err != nil {
		return err
	}

	if err := client.ApplyClusterRoleBinding(&rbac.ClusterRoleBinding{
		ObjectMeta: mach.ObjectMeta{
			Name: "boom-clusterrolebinding",
			Labels: map[string]string{
				"app.kubernetes.io/instance":  "boom",
				"app.kubernetes.io/part-of":   "orbos",
				"app.kubernetes.io/component": "boom",
			},
		},

		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "boom-clusterrole",
		},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      "boom",
			Namespace: "caos-system",
		}},
	}); err != nil {
		return err
	}

	if err := client.ApplyDeployment(&apps.Deployment{
		ObjectMeta: mach.ObjectMeta{
			Name:      "boom",
			Namespace: "caos-system",
			Labels: map[string]string{
				"app.kubernetes.io/instance":   "boom",
				"app.kubernetes.io/part-of":    "orbos",
				"app.kubernetes.io/component":  "boom",
				"app.kubernetes.io/managed-by": "boom.caos.ch",
			},
		},
		Spec: apps.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &mach.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance":  "boom",
					"app.kubernetes.io/part-of":   "orbos",
					"app.kubernetes.io/component": "boom",
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: mach.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/instance":  "boom",
						"app.kubernetes.io/part-of":   "orbos",
						"app.kubernetes.io/component": "boom",
					},
				},
				Spec: core.PodSpec{
					ServiceAccountName: "boom",
					Containers: []core.Container{{
						Name:            "boom",
						ImagePullPolicy: core.PullIfNotPresent,
						Image:           fmt.Sprintf("%s/caos/orbos:%s", imageRegistry, version),
						Command:         []string{"/orbctl", "takeoff", "boom", "-f", "/secrets/orbconfig"},
						Args:            []string{},
						Ports: []core.ContainerPort{{
							Name:          "metrics",
							ContainerPort: 2112,
							Protocol:      "TCP",
						}},
						VolumeMounts: []core.VolumeMount{{
							Name:      "orbconfig",
							ReadOnly:  true,
							MountPath: "/secrets",
						}},
						Resources: core.ResourceRequirements(*resources),
					}},
					NodeSelector: nodeselector,
					Tolerations:  tolerations.K8s(),
					Volumes: []core.Volume{{
						Name: "orbconfig",
						VolumeSource: core.VolumeSource{
							Secret: &core.SecretVolumeSource{
								SecretName: "caos",
							},
						},
					}},
					TerminationGracePeriodSeconds: int64Ptr(10),
				},
			},
		},
	}); err != nil {
		return err
	}
	monitor.WithFields(map[string]interface{}{
		"version": version,
	}).Debug("Boom deployment ensured")

	if err := client.ApplyService(&core.Service{
		ObjectMeta: mach.ObjectMeta{
			Name:      "boom",
			Namespace: "caos-system",
			Labels: map[string]string{
				"app.kubernetes.io/instance":   "boom",
				"app.kubernetes.io/part-of":    "orbos",
				"app.kubernetes.io/component":  "boom",
				"app.kubernetes.io/managed-by": "boom.caos.ch",
			},
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{{
				Name:       "metrics",
				Protocol:   "TCP",
				Port:       2112,
				TargetPort: intstr.FromInt(2112),
			}},
			Selector: map[string]string{
				"app.kubernetes.io/instance":  "boom",
				"app.kubernetes.io/part-of":   "orbos",
				"app.kubernetes.io/component": "boom",
			},
			Type: core.ServiceTypeClusterIP,
		},
	}); err != nil {
		return err
	}
	monitor.Debug("Boom service ensured")

	return nil
}

func EnsureOrbiterArtifacts(
	monitor mntr.Monitor,
	client *Client,
	orbiterversion string,
	imageRegistry string) error {

	monitor.WithFields(map[string]interface{}{
		"orbiter": orbiterversion,
	}).Debug("Ensuring orbiter artifacts")

	if orbiterversion == "" {
		return nil
	}

	if err := client.ApplyDeployment(&apps.Deployment{
		ObjectMeta: mach.ObjectMeta{
			Name:      "orbiter",
			Namespace: "caos-system",
			Labels: map[string]string{
				"app.kubernetes.io/instance":   "orbiter",
				"app.kubernetes.io/part-of":    "orbos",
				"app.kubernetes.io/component":  "orbiter",
				"app.kubernetes.io/managed-by": "orbiter.caos.ch",
			},
		},
		Spec: apps.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &mach.LabelSelector{
				MatchLabels: map[string]string{
					"app": "orbiter",
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: mach.ObjectMeta{
					Labels: map[string]string{
						"app": "orbiter",
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{{
						Name:            "orbiter",
						ImagePullPolicy: core.PullIfNotPresent,
						Image:           fmt.Sprintf("%s/caos/orbos:%s", imageRegistry, orbiterversion),
						Command:         []string{"/orbctl", "--orbconfig", "/etc/orbiter/orbconfig", "takeoff", "orbiter", "--recur", "--ingestion="},
						VolumeMounts: []core.VolumeMount{{
							Name:      "keys",
							ReadOnly:  true,
							MountPath: "/etc/orbiter",
						}},
						Ports: []core.ContainerPort{{
							Name:          "metrics",
							ContainerPort: 9000,
						}},
						Resources: core.ResourceRequirements{
							Limits: core.ResourceList{
								"cpu":    resource.MustParse("500m"),
								"memory": resource.MustParse("500Mi"),
							},
							Requests: core.ResourceList{
								"cpu":    resource.MustParse("250m"),
								"memory": resource.MustParse("250Mi"),
							},
						},
					}},
					Volumes: []core.Volume{{
						Name: "keys",
						VolumeSource: core.VolumeSource{
							Secret: &core.SecretVolumeSource{
								SecretName: "caos",
								Optional:   boolPtr(false),
							},
						},
					}},
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/master": "",
					},
					Tolerations: []core.Toleration{{
						Key:      "node-role.kubernetes.io/master",
						Effect:   "NoSchedule",
						Operator: "Exists",
					}},
				},
			},
		},
	}); err != nil {
		return err
	}
	monitor.WithFields(map[string]interface{}{
		"version": orbiterversion,
	}).Debug("Orbiter deployment ensured")

	if err := client.ApplyService(&core.Service{
		ObjectMeta: mach.ObjectMeta{
			Name:      "orbiter",
			Namespace: "caos-system",
			Labels: map[string]string{
				"app.kubernetes.io/instance":   "orbiter",
				"app.kubernetes.io/part-of":    "orbos",
				"app.kubernetes.io/component":  "orbiter",
				"app.kubernetes.io/managed-by": "orbiter.caos.ch",
			},
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{{
				Name:       "metrics",
				Protocol:   "TCP",
				Port:       9000,
				TargetPort: intstr.FromInt(9000),
			}},
			Selector: map[string]string{
				"app": "orbiter",
			},
			Type: core.ServiceTypeClusterIP,
		},
	}); err != nil {
		return err
	}
	monitor.Debug("Orbiter service ensured")

	patch := `
{
  "spec": {
    "template": {
      "spec": {
        "affinity": {
          "podAntiAffinity": {
            "preferredDuringSchedulingIgnoredDuringExecution": [{
              "weight": 100,
              "podAffinityTerm": {
                "topologyKey": "kubernetes.io/hostname"
              }
            }]
          }
        }
      }
    }
  }
}`
	if err := client.PatchDeployment("kube-system", "coredns", patch); err != nil {
		return err
	}

	monitor.Debug("CoreDNS deployment patched")
	return nil
}

func int32Ptr(i int32) *int32 { return &i }
func int64Ptr(i int64) *int64 { return &i }
func boolPtr(b bool) *bool    { return &b }
