// +build test integration

package core

import (
	"context"
	"os"

	"github.com/caos/orbiter/internal/kinds/clusters/core/infra"
	"github.com/caos/orbiter/internal/kinds/providers/core"
	"github.com/caos/orbiter/internal/kinds/providers/gce"
	gceconfig "github.com/caos/orbiter/internal/kinds/providers/gce/config"
	gcetypes "github.com/caos/orbiter/internal/kinds/providers/gce/config/api"
	"github.com/caos/orbiter/internal/kinds/providers/gce/api"
	"github.com/caos/orbiter/internal/kinds/providers/gce/resourceservices/instance"
	logcontext "github.com/caos/orbiter/logging/context"
	"github.com/caos/orbiter/logging/stdlib"
	"github.com/spf13/viper"
)

type gceProvider struct {
	config  *viper.Viper
	secrets *viper.Viper
}

func Gce(config *viper.Viper, secrets *viper.Viper) Provider {
	return &gceProvider{config, secrets}
}

func (g *gceProvider) Assemble(operatorID string, configuredPools []string, configuredLoadBalancers []*LoadBalancer) (infra.Provider, core.ComputesService, interface{}, error) {

	pools := make(map[string]*gcetypes.Pool)
	for _, pool := range configuredPools {
		pools[pool] = &gcetypes.Pool{
			MinCPUCores: 1,
			MinMemoryGB: 1,
			StorageGB:   15,
		}
	}

	lbs := make(map[string]*gcetypes.LoadBalancer)
	for _, lb := range configuredLoadBalancers {
		lbs[lb.Name] = &gcetypes.LoadBalancer{
			Pools:    lb.Pools,
			Ports:    []int64{700},
			External: true,
			Protocol: gcetypes.TCP,
			HealthChecks: &gcetypes.HealthChecks{
				Path: "/healthz",
				Port: 700,
			},
		}
	}

	ctx := context.Background()

	assembler := gceconfig.New(ctx, g.config, map[string]interface{}{
		"operatorId":    operatorID,
		"pools":         pools,
		"loadbalancers": lbs,
	}, g.secrets)
	assembly, err := assembler.Assemble()
	if err != nil {
		return nil, nil, nil, err
	}

	logger := logcontext.Add(stdlib.New(os.Stdout)).Verbose()
	computesSvc := instance.NewInstanceService(logger, assembly, &api.Caller{
		Ctx: ctx,
		Cfg: assembly.Config(),
	})

	return gce.New(logger, assembly), computesSvc, assembly, nil
}
