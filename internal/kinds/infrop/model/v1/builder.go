package v1

import (
	"fmt"

	"github.com/caos/infrop/internal/core/operator"
	"github.com/caos/infrop/internal/kinds/clusters/kubernetes"
	"github.com/caos/infrop/internal/kinds/clusters/kubernetes/adapter"
	k8s "github.com/caos/infrop/internal/kinds/clusters/kubernetes/model"
	"github.com/caos/infrop/internal/kinds/infrop/model"

	"github.com/mitchellh/mapstructure"
)

func init() {
	build = func(desired map[string]interface{}, _ *operator.Secrets, dependant interface{}) (model.UserSpec, func(model.Config, map[string]map[string]interface{}) (map[string]operator.Assembler, error)) {

		spec := model.UserSpec{}
		err := mapstructure.Decode(desired, &spec)

		return spec, func(cfg model.Config, deps map[string]map[string]interface{}) (map[string]operator.Assembler, error) {

			if err != nil {
				return nil, err
			}
			subassemblers := make(map[string]operator.Assembler)
			for clusterName, clusterConfig := range deps {

				overwriteDesired := func(cName string) func(map[string]interface{}) {
					return func(des map[string]interface{}) {
						if spec.Verbose {
							des["verbose"] = true
						}
						if spec.Destroyed {
							des["destroyed"] = true
						}
					}
				}(clusterName)
				kind, ok := clusterConfig["kind"]
				if !ok {
					return nil, fmt.Errorf("Desired cluster %+v has no kind field", clusterName)
				}

				kindStr, ok := kind.(string)
				if !ok {
					return nil, fmt.Errorf("Desired cluster kind %v must be of type string", kind)
				}

				path := []string{"deps", clusterName}
				switch kindStr {
				case "infrop.caos.ch/KubernetesCluster":
					subassemblers[clusterName] = kubernetes.New(path, overwriteDesired, adapter.New(k8s.Parameters{
						Logger: cfg.Logger.WithFields(map[string]interface{}{
							"cluster": clusterName,
						}),
						ID:               cfg.ConfigID + clusterName,
						SelfAbsolutePath: path,
						RepoURL:          cfg.NodeagentRepoURL,
						RepoKey:          cfg.NodeagentRepoKey,
						MasterKey:        cfg.Masterkey,
						InfropVersion:    cfg.InfropVersion,
						CurrentFile:      cfg.CurrentFile,
						SecretsFile:      cfg.SecretsFile,
					}))
				default:
					return nil, fmt.Errorf("Unknown cluster kind %s", kindStr)
				}
			}
			return subassemblers, nil
		}
	}
}
