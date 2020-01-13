package wrap

import (
	"github.com/caos/orbiter/internal/core/operator/orbiter"
	"github.com/caos/orbiter/internal/kinds/clusters/core/infra"
	"github.com/caos/orbiter/internal/kinds/loadbalancers/dynamic/model"
	"github.com/caos/orbiter/internal/kinds/providers/core"
)

type cmpSvcLB struct {
	original  core.ComputesService
	dynamic   model.Current
	nodeagent func(infra.Compute) *orbiter.NodeAgentCurrent
}

func ComputesService(svc core.ComputesService, dynamic model.Current, nodeagent func(infra.Compute) *orbiter.NodeAgentCurrent) core.ComputesService {
	return &cmpSvcLB{
		original:  svc,
		dynamic:   dynamic,
		nodeagent: nodeagent,
	}
}

func (i *cmpSvcLB) ListPools() ([]string, error) {
	return i.original.ListPools()
}

func (i *cmpSvcLB) List(poolName string, active bool) (infra.Computes, error) {
	return i.original.List(poolName, active)
}

func (i *cmpSvcLB) Create(poolName string) (infra.Compute, error) {
	cmp, err := i.original.Create(poolName)
	if err != nil {
		return nil, err
	}

	desireFunc := desire(poolName, true, i.dynamic, i.original, i.nodeagent)
	return compute(cmp, desireFunc), desireFunc()
}
