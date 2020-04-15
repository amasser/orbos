package kubernetes

import (
	"fmt"
	"strings"

	"github.com/caos/orbiter/internal/operator/common"
	"github.com/caos/orbiter/internal/operator/orbiter/kinds/clusters/core/infra"
	"github.com/caos/orbiter/mntr"
	core "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
)

type initializedPool struct {
	infra    infra.Pool
	tier     Tier
	desired  Pool
	machines func() ([]*initializedMachine, error)
}

type initializeFunc func(initializedPool, []*initializedMachine) error
type uninitializeMachineFunc func(id string)
type initializeMachineFunc func(machine infra.Machine, pool initializedPool) *initializedMachine

func (i *initializedPool) enhance(initialize initializeFunc) {
	original := i.machines
	i.machines = func() ([]*initializedMachine, error) {
		machines, err := original()
		if err != nil {
			return nil, err
		}
		if err := initialize(*i, machines); err != nil {
			return nil, err
		}
		return machines, nil
	}
}

type initializedMachine struct {
	infra            infra.Machine
	tier             Tier
	reconcile        func() error
	currentNodeagent *common.NodeAgentCurrent
	desiredNodeagent *common.NodeAgentSpec
	currentMachine   *Machine
}

func initialize(
	monitor mntr.Monitor,
	curr *CurrentCluster,
	desired DesiredV0,
	nodeAgentsCurrent map[string]*common.NodeAgentCurrent,
	nodeAgentsDesired map[string]*common.NodeAgentSpec,
	providerPools map[string]map[string]infra.Pool,
	k8s *Client,
	postInit func(machine *initializedMachine)) (
	controlplane initializedPool,
	controlplaneMachines []*initializedMachine,
	workers []initializedPool,
	workerMachines []*initializedMachine,
	initializeMachine initializeMachineFunc,
	uninitializeMachine uninitializeMachineFunc,
	err error) {

	if curr.Machines == nil {
		curr.Machines = make(map[string]*Machine)
	}

	curr.Status = "running"

	initializePool := func(infraPool infra.Pool, desired Pool, tier Tier) initializedPool {
		pool := initializedPool{
			infra:   infraPool,
			tier:    tier,
			desired: desired,
		}
		pool.machines = func() ([]*initializedMachine, error) {
			infraMachines, err := infraPool.GetMachines(true)
			if err != nil {
				return nil, err
			}
			machines := make([]*initializedMachine, len(infraMachines))
			for i, infraMachine := range infraMachines {
				machines[i] = initializeMachine(infraMachine, pool)
				if !machines[i].currentMachine.Online {
					curr.Status = "maintaining"
				}
			}
			return machines, nil
		}
		return pool
	}

	initializeMachine = func(machine infra.Machine, pool initializedPool) *initializedMachine {

		node, getNodeErr := k8s.GetNode(machine.ID())

		current := &Machine{
			Metadata: MachineMetadata{
				Tier:     pool.tier,
				Provider: pool.desired.Provider,
				Pool:     pool.desired.Pool,
			},
		}

		if getNodeErr == nil {
			current.Joined = true
			if !node.Spec.Unschedulable {
				for _, cond := range node.Status.Conditions {
					if cond.Type == v1.NodeReady {
						current.Online = true
						break
					}
				}
			}
		}

		curr.Machines[machine.ID()] = current
		reconcileNode := false
		reconcile := func() error { return nil }
		if node != nil {
			reconcileMonitor := monitor.WithField("node", node.Name)
			poolLabelKey := "orbos.ch/pool"
			if node.Labels[poolLabelKey] != pool.desired.Pool {
				reconcileNode = true
				reconcileMonitor = reconcileMonitor.WithField("label", fmt.Sprintf("%s=%s", poolLabelKey, pool.desired.Pool))
				node.Labels[poolLabelKey] = pool.desired.Pool
			}

			desiredTaints := pool.desired.Taints.ToK8sTaints()
			newTaints := append([]core.Taint{}, desiredTaints...)
			updateTaints := false
		outer:
			for _, existing := range node.Spec.Taints {
				if strings.HasPrefix(existing.Key, "node.kubernetes.io/") {
					newTaints = append(newTaints, existing)
					continue
				}
				for _, des := range desiredTaints {
					if existing.Key == des.Key &&
						existing.Effect == des.Effect &&
						existing.Value == des.Value {
						continue outer
					}
					updateTaints = true
					break
				}
			}
			if updateTaints || len(node.Spec.Taints) != len(newTaints) {
				reconcileNode = true
				node.Spec.Taints = newTaints
				reconcileMonitor = reconcileMonitor.WithField("taints", desiredTaints)
			}

			if reconcileNode {
				reconcile = func() error {
					reconcileMonitor.Info("Reconciling node")
					return k8s.updateNode(node)
				}
			}
		}

		machineMonitor := monitor.WithField("machine", machine.ID())

		naSpec, ok := nodeAgentsDesired[machine.ID()]
		if !ok {
			naSpec = &common.NodeAgentSpec{}
			nodeAgentsDesired[machine.ID()] = naSpec
		}
		naSpec.ChangesAllowed = !pool.desired.UpdatesDisabled

		naCurr, ok := nodeAgentsCurrent[machine.ID()]
		if !ok || naCurr == nil {
			naCurr = &common.NodeAgentCurrent{}
			nodeAgentsCurrent[machine.ID()] = naCurr
		}

		if naSpec.Software == nil {
			naSpec.Software = &common.Software{}
		}

		k8sSoftware := ParseString(desired.Spec.Versions.Kubernetes).DefineSoftware()
		if !naSpec.Software.Defines(k8sSoftware) {
			k8sSoftware.Merge(KubernetesSoftware(naCurr.Software))
			if !naSpec.Software.Contains(k8sSoftware) {
				naSpec.Software.Merge(k8sSoftware)
				machineMonitor.Changed("Kubernetes software desired")
			}
		}

		initMachine := &initializedMachine{
			infra:            machine,
			currentNodeagent: naCurr,
			desiredNodeagent: naSpec,
			tier:             pool.tier,
			reconcile:        reconcile,
			currentMachine:   current,
		}

		postInit(initMachine)

		return initMachine
	}

	for providerName, provider := range providerPools {
	pools:
		for poolName, pool := range provider {
			if desired.Spec.ControlPlane.Provider == providerName && desired.Spec.ControlPlane.Pool == poolName {
				controlplane = initializePool(pool, desired.Spec.ControlPlane, Controlplane)
				controlplaneMachines, err = controlplane.machines()
				if err != nil {
					return controlplane,
						controlplaneMachines,
						workers,
						workerMachines,
						initializeMachine,
						uninitializeMachine,
						err
				}
				continue
			}

			for _, desiredPool := range desired.Spec.Workers {
				if providerName == desiredPool.Provider && poolName == desiredPool.Pool {
					workerPool := initializePool(pool, *desiredPool, Workers)
					workers = append(workers, workerPool)
					initializedWorkerMachines, err := workerPool.machines()
					if err != nil {
						return controlplane,
							controlplaneMachines,
							workers,
							workerMachines,
							initializeMachine,
							uninitializeMachine,
							err
					}
					workerMachines = append(workerMachines, initializedWorkerMachines...)
					continue pools
				}
			}
		}
	}

	for _, machine := range append(controlplaneMachines, workerMachines...) {
		if !machine.currentMachine.Online || !machine.currentMachine.Joined || !machine.currentMachine.NodeAgentIsRunning || !machine.currentMachine.FirewallIsReady {
			curr.Status = "maintaining"
			break
		}
	}

	return controlplane,
		controlplaneMachines,
		workers,
		workerMachines,
		initializeMachine, func(id string) {
			delete(nodeAgentsDesired, id)
			delete(curr.Machines, id)
		}, nil
}
