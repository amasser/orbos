// Code generated by "gen-kindstubs -parentpath=github.com/caos/orbiter/internal/kinds/providers -versions=v1 -kind=orbiter.caos.ch/GCEProvider from file gen.go"; DO NOT EDIT.

package gce

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"

	"github.com/caos/orbiter/internal/core/operator"

	"github.com/caos/orbiter/internal/kinds/providers/gce/adapter"
	"github.com/caos/orbiter/internal/kinds/providers/gce/model"
	v1builder "github.com/caos/orbiter/internal/kinds/providers/gce/model/v1"
)

type Version int

const (
	unknown Version = iota
	v1
)

type Kind struct {
	Kind    string
	Version string
	Spec    map[string]interface{}
	Deps    map[string]map[string]interface{}
}

type assembler struct {
	path      []string
	overwrite func(map[string]interface{})
	builder   adapter.Builder
	built     adapter.Adapter
}

func New(configPath []string, overwrite func(map[string]interface{}), builder adapter.Builder) operator.Assembler {
	return &assembler{configPath, overwrite, builder, nil}
}

func (a *assembler) String() string { return "orbiter.caos.ch/GCEProvider" }
func (a *assembler) BuildContext() ([]string, func(map[string]interface{})) {
	return a.path, a.overwrite
}
func (a *assembler) Ensure(ctx context.Context, secrets *operator.Secrets, ensuredDependencies map[string]interface{}) (interface{}, error) {
	return a.built.Ensure(ctx, secrets, ensuredDependencies)
}
func (a *assembler) Build(serialized map[string]interface{}, nodeagentupdater operator.NodeAgentUpdater, secrets *operator.Secrets, dependant interface{}) (string, string, interface{}, map[string]operator.Assembler, error) {

	kind := &Kind{}
	if err := mapstructure.Decode(serialized, kind); err != nil {
		return "", "", nil, nil, err
	}

	if kind.Kind != "orbiter.caos.ch/GCEProvider" {
		return "", "", nil, nil, fmt.Errorf("Kind %s must be \"orbiter.caos.ch/GCEProvider\"", kind.Kind)
	}

	var spec model.UserSpec
	var subassemblersBuilder func(model.Config, map[string]map[string]interface{}) (map[string]operator.Assembler, error)
	switch kind.Version {
	case v1.String():
		spec, subassemblersBuilder = v1builder.Build(kind.Spec, secrets, dependant)
	default:
		return "", "", nil, nil, fmt.Errorf("Unknown version %s", kind.Version)
	}

	cfg, adapter, err := a.builder.Build(spec, nodeagentupdater)
	if err != nil {
		return "", "", nil, nil, err
	}
	a.built = adapter

	if subassemblersBuilder == nil {
		return kind.Kind, kind.Version, cfg, nil, nil
	}

	subassemblers, err := subassemblersBuilder(cfg, kind.Deps)
	if err != nil {
		return "", "", nil, nil, err
	}

	return kind.Kind, kind.Version, cfg, subassemblers, nil
}
