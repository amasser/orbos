package v1

import (
	"fmt"

	"github.com/mitchellh/mapstructure"

	"github.com/caos/infrop/internal/core/operator"
	"github.com/caos/infrop/internal/kinds/clusters/core/infra"
	"github.com/caos/infrop/internal/kinds/clusters/kubernetes/model"
	"github.com/caos/infrop/internal/kinds/providers/gce"
	gceadapter "github.com/caos/infrop/internal/kinds/providers/gce/adapter"
	"github.com/caos/infrop/internal/kinds/providers/static"
	staticadapter "github.com/caos/infrop/internal/kinds/providers/static/adapter"

	"github.com/caos/infrop/internal/core/logging"
	"github.com/caos/infrop/internal/kinds/clusters/kubernetes/edge/k8s"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	mach "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type subBuilder struct {
	assembler operator.Assembler
	desired   map[string]interface{}
	current   map[string]interface{}
}

func init() {
	build = func(desired map[string]interface{}, secrets *operator.Secrets, _ interface{}) (model.UserSpec, func(model.Config, map[string]map[string]interface{}) (map[string]operator.Assembler, error)) {

		spec := model.UserSpec{}
		err := mapstructure.Decode(desired, &spec)

		return spec, func(cfg model.Config, deps map[string]map[string]interface{}) (map[string]operator.Assembler, error) {

			if spec.Versions.Infrop != "" {
				if ensureErr := ensureArtifacts(cfg.Params.Logger, secrets, cfg.Params.ID, cfg.Params.RepoURL, cfg.Params.RepoKey, cfg.Params.MasterKey, spec.Versions.Infrop); err != nil {
					return nil, ensureErr
				}
			}

			if err != nil {
				return nil, err
			}

			subassemblers := make(map[string]operator.Assembler)
			for providerName, providerConfig := range deps {

				providerPath := []string{"deps", providerName}
				generalOverwriteSpec := func(providerName string) func(des map[string]interface{}) {
					return func(des map[string]interface{}) {
						if spec.Verbose {
							des["verbose"] = true
						}
					}
				}(providerName)

				kind, ok := providerConfig["kind"]
				if !ok {
					return nil, fmt.Errorf("Spec provider %+v has no kind field", providerName)
				}
				kindStr, ok := kind.(string)
				if !ok {
					return nil, fmt.Errorf("Spec provider kind %v must be of type string", kind)
				}

				providerlogger := cfg.Params.Logger.WithFields(map[string]interface{}{
					"provider": providerName,
				})
				providerID := cfg.Params.ID + providerName
				switch kindStr {
				case "infrop.caos.ch/GCEProvider":
					var lbs map[string]*infra.Ingress

					if !spec.Destroyed && spec.ControlPlane.Provider == providerName {
						lbs = map[string]*infra.Ingress{
							"kubeapi": &infra.Ingress{
								Pools:            []string{spec.ControlPlane.Pool},
								HealthChecksPath: "/healthz",
							},
						}
					}
					subassemblers[providerName] = gce.New(providerPath, generalOverwriteSpec, gceadapter.New(providerlogger, providerID, lbs, nil, ""))
				case "infrop.caos.ch/StaticProvider":
					updatesDisabled := make([]string, 0)
					for _, pool := range spec.Workers {
						if pool.UpdatesDisabled {
							updatesDisabled = append(updatesDisabled, pool.Pool)
						}
					}

					if spec.ControlPlane.UpdatesDisabled {
						updatesDisabled = append(updatesDisabled, spec.ControlPlane.Pool)
					}

					subassemblers[providerName] = static.New(providerPath, generalOverwriteSpec, staticadapter.New(providerlogger, providerID, "/healthz", updatesDisabled, cfg.NodeAgent))
				default:
					return nil, fmt.Errorf("Provider of kind %s is unknown", kindStr)
				}
			}

			return subassemblers, nil
		}
	}
}

func ensureArtifacts(logger logging.Logger, secrets *operator.Secrets, secretsNamespace, repourl, repokey, masterkey, infropversion string) error {

	kc, err := secrets.Read(secretsNamespace + "_kubeconfig")
	if err != nil {
		return nil
	}

	kcStr := string(kc)
	client := k8s.New(logger, &kcStr)

	if err := client.ApplyNamespace(&core.Namespace{
		ObjectMeta: mach.ObjectMeta{
			Name: "caos-system",
			Labels: map[string]string{
				"name": "caos-system",
			},
		},
	}); err != nil {
		return err
	}

	if err := client.ApplySecret(&core.Secret{
		ObjectMeta: mach.ObjectMeta{
			Name:      "caos",
			Namespace: "caos-system",
		},
		StringData: map[string]string{
			"repokey":   repokey,
			"masterkey": masterkey,
		},
	}); err != nil {
		return err
	}

	if err := client.ApplyDeployment(&apps.Deployment{
		ObjectMeta: mach.ObjectMeta{
			Name:      "infrop",
			Namespace: "caos-system",
		},
		Spec: apps.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &mach.LabelSelector{
				MatchLabels: map[string]string{
					"app": "infrop",
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: mach.ObjectMeta{
					Labels: map[string]string{
						"app": "infrop",
					},
				},
				Spec: core.PodSpec{
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/master": "",
					},
					Tolerations: []core.Toleration{{
						Key:      "node-role.kubernetes.io/master",
						Operator: "Equal",
						Value:    "",
						Effect:   "NoSchedule",
					}},
					// TODO: Remove before open sourcing #39
					ImagePullSecrets: []core.LocalObjectReference{{
						Name: "infropregistry",
					}},
					Containers: []core.Container{{
						Name:            "infrop",
						ImagePullPolicy: core.PullAlways,
						Image:           fmt.Sprintf("docker.pkg.github.com/caos/infrop/infrop:%s", infropversion),
						Command:         []string{"/artifacts/infrop", "--recur", "--repourl", repourl},
						VolumeMounts: []core.VolumeMount{{
							Name:      "keys",
							ReadOnly:  true,
							MountPath: "/etc/infrop",
						}},
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
				},
			},
		},
	}); err != nil {
		return err
	}

	// TODO: Apply toolsop
	return nil
}

func int32Ptr(i int32) *int32 { return &i }
func boolPtr(b bool) *bool    { return &b }
