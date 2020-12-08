package centos

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	"github.com/caos/orbos/internal/operator/common"
	"github.com/caos/orbos/internal/operator/nodeagent"
	"github.com/caos/orbos/mntr"
)

func Ensurer(monitor mntr.Monitor, ignore []string) nodeagent.FirewallEnsurer {
	return nodeagent.FirewallEnsurerFunc(func(desired common.Firewall) (common.FirewallCurrent, func() error, error) {
		ensurers := make([]func() error, 0)
		current := make(common.FirewallCurrent, 0)

		if desired.Zones == nil {
			desired.Zones = make(map[string]*common.Zone, 0)
		}

		for name, _ := range desired.Zones {
			currentZone, ensureFunc, err := ensureZone(monitor, name, desired, ignore)
			if err != nil {
				return current, nil, err
			}
			current = append(current, currentZone)
			if ensureFunc != nil {
				ensurers = append(ensurers, ensureFunc)
			}
		}

		cmd := exec.Command("systemctl", "is-active", "firewalld")
		if monitor.IsVerbose() {
			fmt.Println(strings.Join(cmd.Args, " "))
			cmd.Stdout = os.Stdout
		}

		if cmd.Run() != nil || len(ensurers) == 0 {
			monitor.Debug("Not changing firewall")
			return current, nil, nil
		}

		current.Sort()

		return current, func() error {
			monitor.Debug("Ensuring firewall")
			for _, ensurer := range ensurers {
				if err := ensurer(); err != nil {
					return err
				}
			}
			return nil
		}, nil
	})
}

func ensureZone(monitor mntr.Monitor, zoneName string, desired common.Firewall, ignore []string) (*common.ZoneDesc, func() error, error) {
	current := &common.ZoneDesc{
		Name:       zoneName,
		Interfaces: []string{},
		Services:   []*common.Service{},
		FW:         []*common.Allowed{},
	}

	sources, err := getSources(monitor, zoneName)
	if err != nil {
		return current, nil, err
	}
	current.Sources = sources

	ensureMasquerade, err := getEnsureMasquerade(monitor, zoneName, current, desired)
	if err != nil {
		return current, nil, err
	}

	addPorts, removePorts, err := getAddAndRemovePorts(monitor, zoneName, current, desired.Ports(zoneName), ignore)
	if err != nil {
		return current, nil, err
	}

	ensureIfaces, err := getEnsureInterfaces(monitor, zoneName, current, desired)
	if err != nil {
		return current, nil, err
	}

	addSources, removeSources, err := getAddAndRemoveSources(monitor, zoneName, current, desired)
	if err != nil {
		return current, nil, err
	}

	monitor.WithFields(map[string]interface{}{
		"open":  strings.Join(addPorts, ";"),
		"close": strings.Join(removePorts, ";"),
	}).Debug("firewall changes determined")

	if (addPorts == nil || len(addPorts) == 0) &&
		(removePorts == nil || len(removePorts) == 0) &&
		(addSources == nil || len(addSources) == 0) &&
		(removeSources == nil || len(removeSources) == 0) &&
		(ensureIfaces == nil || len(ensureIfaces) == 0) {
		return current, nil, nil
	}

	zoneNameCopy := zoneName
	return current, func() error {
		if ensureMasquerade != "" {
			monitor.Debug(fmt.Sprintf("Ensuring part of firewall with %s in zone %s", ensureMasquerade, zoneNameCopy))
			if err := ensure(monitor, []string{ensureMasquerade}, zoneNameCopy); err != nil {
				return err
			}
		}

		monitor.Debug(fmt.Sprintf("Ensuring part of firewall with %s in zone %s", ensureIfaces, zoneNameCopy))
		if err := ensure(monitor, ensureIfaces, zoneNameCopy); err != nil {
			return err
		}

		monitor.Debug(fmt.Sprintf("Ensuring part of firewall with %s in zone %s", addSources, zoneNameCopy))
		if err := ensure(monitor, addSources, zoneNameCopy); err != nil {
			return err
		}

		monitor.Debug(fmt.Sprintf("Ensuring part of firewall with %s in zone %s", removeSources, zoneNameCopy))
		if err := ensure(monitor, removeSources, zoneNameCopy); err != nil {
			return err
		}

		monitor.Debug(fmt.Sprintf("Ensuring part of firewall with %s in zone %s", addPorts, zoneNameCopy))
		if err := ensure(monitor, addPorts, zoneNameCopy); err != nil {
			return err
		}

		monitor.Debug(fmt.Sprintf("Ensuring part of firewall with %s in zone %s", removePorts, zoneNameCopy))
		return ensure(monitor, removePorts, zoneNameCopy)
	}, nil
}

func ensure(monitor mntr.Monitor, changes []string, zone string) error {
	if changes == nil || len(changes) == 0 {
		return nil
	}

	errBuf := new(bytes.Buffer)
	defer errBuf.Reset()

	cmd := exec.Command("systemctl", "enable", "firewalld")
	cmd.Stderr = errBuf

	fullCmd := strings.Join(cmd.Args, " ")
	if monitor.IsVerbose() {
		fmt.Println(fullCmd)
		cmd.Stdout = os.Stdout
	}

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "running %s failed with stderr %s", fullCmd, errBuf.String())
	}

	errBuf.Reset()
	cmd = exec.Command("systemctl", "start", "firewalld")
	cmd.Stderr = errBuf

	fullCmd = strings.Join(cmd.Args, " ")
	if monitor.IsVerbose() {
		fmt.Println(fullCmd)
		cmd.Stdout = os.Stdout
	}

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "running %s failed with stderr %s", fullCmd, errBuf.String())
	}

	return changeFirewall(monitor, changes, zone)
}

func changeFirewall(monitor mntr.Monitor, changes []string, zone string) (err error) {

	changesMonitor := monitor.WithField("changes", strings.Join(changes, ";"))
	changesMonitor.Debug("Changing firewall")

	defer func() {
		if err == nil {
			changesMonitor.Debug("firewall changed")
		} else {
			changesMonitor.Error(err)
		}
	}()

	errBuf := new(bytes.Buffer)
	defer errBuf.Reset()
	if len(changes) == 0 {
		return nil
	}

	errBuf.Reset()
	args := strings.Join([]string{"sudo firewall-cmd --permanent --zone", zone, strings.Join(changes, " ")}, " ")
	cmd := exec.Command("/bin/sh", append([]string{"-c"}, args)...)
	cmd.Stderr = errBuf

	fullCmd := strings.Join(cmd.Args, " ")
	if monitor.IsVerbose() {
		fmt.Println(fullCmd)
		cmd.Stdout = os.Stdout
	}

	if err := errors.Wrapf(cmd.Run(), "running %s failed with stderr %s", fullCmd, errBuf.String()); err != nil {
		return err
	}

	return reloadFirewall(monitor)
}

func reloadFirewall(monitor mntr.Monitor) error {
	errBuf := new(bytes.Buffer)
	errBuf.Reset()
	cmd := exec.Command("/bin/sh", "-c", "sudo firewall-cmd --reload")
	cmd.Stderr = errBuf
	if monitor.IsVerbose() {
		fmt.Println(strings.Join(cmd.Args, " "))
		cmd.Stdout = os.Stdout
	}

	return errors.Wrapf(cmd.Run(), "running firewall-cmd --reload failed with stderr %s", errBuf.String())
}

func listFirewall(monitor mntr.Monitor, zone string, arg string) ([]string, error) {
	outBuf := new(bytes.Buffer)
	defer outBuf.Reset()
	errBuf := new(bytes.Buffer)
	defer errBuf.Reset()

	cmd := exec.Command("/bin/sh", "-c", "sudo firewall-cmd --zone "+zone+" "+arg)
	cmd.Stderr = errBuf
	cmd.Stdout = outBuf

	if err := cmd.Run(); err != nil {
		return nil, errors.Wrapf(err, "running firewall-cmd %s in order to list firewall failed with stderr %s", arg, errBuf.String())
	}

	stdout := outBuf.String()
	if monitor.IsVerbose() {
		fmt.Println(strings.Join(cmd.Args, " "))
		fmt.Println(stdout)
	}

	return strings.Fields(stdout), nil
}
