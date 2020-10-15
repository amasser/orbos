package cs

import (
	"fmt"

	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/core/infra"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/loadbalancers/dynamic"
	"github.com/cloudscale-ch/cloudscale-go-sdk"
	"github.com/pkg/errors"
)

func queryServers(context *context, current *Current, loadbalancing map[string][]*dynamic.VIP, ensureNodeAgent func(m infra.Machine) error) ([]func() error, error) {

	pools, err := context.machinesService.machines()
	if err != nil {
		return nil, err
	}

	var ensureServers []func() error
	for poolName, machines := range pools {
		for idx := range machines {
			mach := machines[idx]
			ensureServers = append(ensureServers, func(poolName string, m *machine) func() error {
				return func() error {
					return ensureServer(context, current, loadbalancing, poolName, m, ensureNodeAgent)
				}
			}(poolName, mach))
		}
	}
	return ensureServers, nil
}

func ensureServer(context *context, current *Current, loadbalancing map[string][]*dynamic.VIP, poolName string, machine *machine, ensureNodeAgent func(m infra.Machine) error) (err error) {
	defer func() {
		if err == nil {
			err = ensureOS(ensureNodeAgent, machine, loadbalancing, current, context, poolName)
		}
	}()

	_, isExternal := loadbalancing[poolName]
	if context.machinesService.oneoff {
		isExternal = true
	}
	// Always use external ips
	isExternal = true
	hasPublicInterface := false
	var privateInterfaces []cloudscale.Interface
	for idx := range machine.server.Interfaces {
		interf := machine.server.Interfaces[idx]
		if interf.Type == "public" {
			hasPublicInterface = true
		} else {
			privateInterfaces = append(privateInterfaces, interf)
		}
	}

	var updateInterfaces []cloudscale.InterfaceRequest
	if isExternal && !hasPublicInterface {
		updateInterfaces = append(interfaces(machine.server.Interfaces).toRequests(), cloudscale.InterfaceRequest{Network: "public"})
	}

	if !isExternal && hasPublicInterface {
		updateInterfaces = interfaces(privateInterfaces).toRequests()
	}

	if updateInterfaces == nil {
		return nil
	}
	return updateServer(context, machine.server, &updateInterfaces)

}

func ensureOS(ensureNodeAgent func(m infra.Machine) error, machine *machine, loadbalancing map[string][]*dynamic.VIP, current *Current, context *context, poolName string) error {
	if err := ensureNodeAgent(machine); err != nil {
		return err
	}

	return configureFirewall(machine, loadbalancing, current, context, poolName)
}

// TODO: Move this capabilities to where they belong
func configureFirewall(machine *machine, loadbalancing map[string][]*dynamic.VIP, current *Current, context *context, poolName string) error {
	if _, err := machine.Execute(nil, "ifconfig -a | grep dummy"); err != nil {
		cmd, err := addDummyIPCommand(hostedVIPs(loadbalancing, machine, current))
		if err != nil {
			return err
		}

		context.monitor.WithField("cmd", cmd).Info("Executing")
		if _, err = machine.Execute(nil, cmd); err != nil {
			return err
		}
	}
	masq, _ := machine.Execute(nil, "firewall-cmd --list-all | grep 'masquerade: yes'")
	if len(masq) == 0 {
		cmd := "firewall-cmd --add-masquerade --permanent && firewall-cmd --reload"
		context.monitor.WithField("cmd", cmd).Info("Executing")
		if _, err := machine.Execute(nil, cmd); err != nil {
			return err
		}
	}

	return nil
}

func updateServer(context *context, server *cloudscale.Server, interfaces *[]cloudscale.InterfaceRequest) error {
	return context.client.Servers.Update(context.ctx, server.UUID, &cloudscale.ServerUpdateRequest{
		TaggedResourceRequest: cloudscale.TaggedResourceRequest{Tags: server.Tags},
		Name:                  server.Name,
		Status:                server.Status,
		Flavor:                server.Flavor.Name,
		Interfaces:            interfaces,
	})
}

type interfaces []cloudscale.Interface

func (i interfaces) toRequests() []cloudscale.InterfaceRequest {
	var requests []cloudscale.InterfaceRequest
	for idx := range i {
		interf := i[idx]
		addr := addresses(interf.Addresses).toRequest()
		requests = append(requests, cloudscale.InterfaceRequest{
			Network:   interf.Network.UUID,
			Addresses: &addr,
		})
	}
	return requests
}

type addresses []cloudscale.Address

func (a addresses) toRequest() []cloudscale.AddressRequest {
	var requests []cloudscale.AddressRequest
	for idx := range a {
		addr := a[idx]
		requests = append(requests, cloudscale.AddressRequest{
			Subnet:  addr.Subnet.UUID,
			Address: addr.Address,
		})
	}
	return requests
}

func addDummyIPCommand(ips []string) (string, error) {

	if len(ips) == 0 {
		return "true", errors.New("No ips")
	}

	cmd := "ip link add dummy1 type dummy"
	for idx := range ips {
		ip := ips[idx]
		if ip == "" {
			return "true", errors.New("void ip")
		}
		cmd += fmt.Sprintf(" && ip addr add %s/32 dev dummy1", ip)
	}

	return cmd, nil
}
