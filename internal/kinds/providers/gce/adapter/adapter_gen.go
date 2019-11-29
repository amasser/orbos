// Code generated by "gen-kindstubs -parentpath=github.com/caos/infrop/internal/kinds/providers -versions=v1 -kind=infrop.caos.ch/GCEProvider from file gen.go"; DO NOT EDIT.

package adapter

import (
	"context"

	"github.com/caos/infrop/internal/core/operator"
	"github.com/caos/infrop/internal/kinds/providers/gce/model"
)

type Builder interface {
	Build(model.UserSpec, operator.NodeAgentUpdater) (model.Config, Adapter, error)
}

type builderFunc func(model.UserSpec, operator.NodeAgentUpdater) (model.Config, Adapter, error)

func (b builderFunc) Build(spec model.UserSpec, nodeagent operator.NodeAgentUpdater) (model.Config, Adapter, error) {
	return b(spec, nodeagent)
}

type Adapter interface {
	Ensure(context.Context, *operator.Secrets, map[string]interface{}) (*model.Current, error)
}

type adapterFunc func(context.Context, *operator.Secrets, map[string]interface{}) (*model.Current, error)

func (a adapterFunc) Ensure(ctx context.Context, secrets *operator.Secrets, ensuredDependencies map[string]interface{}) (*model.Current, error) {
	return a(ctx, secrets, ensuredDependencies)
}
