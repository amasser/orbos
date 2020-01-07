// Code generated by "gen-kindstubs -parentpath=github.com/caos/orbiter/internal/kinds -versions=v0 -kind=orbiter.caos.ch/Orbiter from file gen.go"; DO NOT EDIT.

package v0

import (
	"errors"

	"github.com/caos/orbiter/internal/core/operator"
	"github.com/caos/orbiter/internal/kinds/orbiter/model"
)

var build func(map[string]interface{}, *operator.Secrets, interface{}) (model.UserSpec, func(model.Config, []map[string]interface{}) (map[string]operator.Assembler, error))

func Build(spec map[string]interface{}, secrets *operator.Secrets, dependant interface{}) (model.UserSpec, func(cfg model.Config, deps []map[string]interface{}) (map[string]operator.Assembler, error)) {
	if build != nil {
		return build(spec, secrets, dependant)
	}
	return model.UserSpec{}, func(_ model.Config, _ []map[string]interface{}) (map[string]operator.Assembler, error) {
		return nil, errors.New("Version v0 for kind orbiter.caos.ch/Orbiter is not yet supported")
	}
}
