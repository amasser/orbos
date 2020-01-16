package v0

import (
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/caos/orbiter/internal/core/operator/orbiter"
"github.com/caos/orbiter/internal/core/operator/common"
	"github.com/caos/orbiter/internal/kinds/loadbalancers/dynamic"
	dynamiclbadapter "github.com/caos/orbiter/internal/kinds/loadbalancers/dynamic/adapter"
	"github.com/caos/orbiter/internal/kinds/loadbalancers/external"
	externallbadapter "github.com/caos/orbiter/internal/kinds/loadbalancers/external/adapter"
	"github.com/caos/orbiter/internal/kinds/providers/static/model"
)

func init() {
	build = func(desired map[string]interface{}, _ *orbiter.Secrets, _ interface{}) (model.UserSpec, func(model.Config) ([]orbiter.Assembler, error)) {

		kind := struct {
			Spec model.UserSpec
			Deps struct {
				Loadbalancing map[string]interface{}
			}
		}{}
		err := mapstructure.Decode(desired, &kind)

		return kind.Spec, func(cfg model.Config) ([]orbiter.Assembler, error) {

			if err != nil {
				return nil, err
			}

			generalOverwriteSpec := func(des map[string]interface{}) {
				if kind.Spec.Verbose {
					des["verbose"] = true
				}
			}

			depPath := []string{"deps", "loadbalancing"}
			depKind := kind.Deps.Loadbalancing["kind"]

			switch depKind {
			case "orbiter.caos.ch/ExternalLoadBalancer":
				return []orbiter.Assembler{external.New(depPath, generalOverwriteSpec, externallbadapter.New())}, nil
			case "orbiter.caos.ch/DynamicLoadBalancer":
				return []orbiter.Assembler{dynamic.New(depPath, generalOverwriteSpec, dynamiclbadapter.New(kind.Spec.RemoteUser))}, nil
			default:
				return nil, errors.Errorf("unknown dependency type %s", depKind)
			}
		}
	}
}
