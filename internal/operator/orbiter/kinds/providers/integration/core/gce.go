// +build test integration

package core

import (
	"context"
	"os"

	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/core/infra"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/providers/core"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/providers/gce"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/providers/gce/api"
	gceconfig "github.com/caos/orbos/internal/operator/orbiter/kinds/providers/gce/config"
	gcetypes "github.com/caos/orbos/internal/operator/orbiter/kinds/providers/gce/config/api"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/providers/gce/resourceservices/instance"
	logcontext "github.com/caos/orbos/logging/context"
	"github.com/caos/orbos/logging/stdlib"
	"github.com/spf13/viper"
)

type gceProvider struct {
	config  *viper.Viper
	secrets *viper.Viper
}

func Gce(config *viper.Viper, secrets *viper.Viper) Provider {
	return &gceProvider{config, secrets}
}

func (g *gceProvider) Assemble(operatorID string, configuredPools []string, configuredLoadBalancers []*LoadBalancer) (infra.Provider, core.MachinesService, interface{}, error) {

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

	monitor := logcontext.Add(stdlib.New(os.Stdout)).Verbose()
	machinesSvc := instance.NewInstanceService(monitor, assembly, &api.Caller{
		Ctx: ctx,
		Cfg: assembly.Config(),
	})

	return gce.New(monitor, assembly), machinesSvc, assembly, nil
}
