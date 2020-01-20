package v0

import (
	"errors"

	"github.com/mitchellh/mapstructure"

	"github.com/caos/orbiter/internal/operator/orbiter"
"github.com/caos/orbiter/internal/operator/common"
	"github.com/caos/orbiter/internal/operator/orbiter/kinds/providers/gce/model"
)

func init() {

	build = func(desired map[string]interface{}, _ *orbiter.Secrets, _ interface{}) (model.UserSpec, func(model.Config, []map[string]interface{}) (map[string]orbiter.Assembler, error)) {

		spec := model.UserSpec{}
		err := mapstructure.Decode(desired, &spec)

		return spec, func(cfg model.Config, deps []map[string]interface{}) (map[string]orbiter.Assembler, error) {

			if err != nil {
				return nil, err
			}

			if len(deps) > 0 {
				return nil, errors.New("GCE provider does not take dependencies")
			}

			return nil, nil
		}
	}
}
