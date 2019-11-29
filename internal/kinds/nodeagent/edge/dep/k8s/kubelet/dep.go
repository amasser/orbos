package kubelet

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	"github.com/caos/infrop/internal/core/logging"
	"github.com/caos/infrop/internal/core/operator"
	"github.com/caos/infrop/internal/kinds/nodeagent/adapter"
	"github.com/caos/infrop/internal/kinds/nodeagent/edge/dep"
	"github.com/caos/infrop/internal/kinds/nodeagent/edge/dep/k8s"
	"github.com/caos/infrop/internal/kinds/nodeagent/edge/dep/middleware"
)

type Installer interface {
	isKubelet()
	adapter.Installer
}

type kubeletDep struct {
	os      dep.OperatingSystem
	logger  logging.Logger
	common  *k8s.Common
	systemd *dep.SystemD
}

func New(logger logging.Logger, os dep.OperatingSystem, manager *dep.PackageManager, systemd *dep.SystemD) Installer {
	return &kubeletDep{os, logger, k8s.New(os, manager, "kubelet"), systemd}
}

func (kubeletDep) isKubelet() {}

func (kubeletDep) Is(other adapter.Installer) bool {
	_, ok := middleware.Unwrap(other).(Installer)
	return ok
}

func (k kubeletDep) String() string { return "Kubelet" }

func (*kubeletDep) Equals(other adapter.Installer) bool {
	_, ok := other.(*kubeletDep)
	return ok
}

func (k *kubeletDep) Current() (operator.Package, error) {
	return k.common.Current()
}

func (k *kubeletDep) Ensure(remove operator.Package, install operator.Package) (bool, error) {
	// TODO: Idempotency: defer uninstall command if an error occurs

	if err := k.common.Ensure(remove, install); err != nil {
		return false, err
	}

	if err := k.systemd.Enable("kubelet"); err != nil {
		return false, err
	}

	if k.os != dep.CentOS {
		return false, nil
	}

	var errBuf bytes.Buffer
	cmd := exec.Command("setenforce", "0")
	cmd.Stderr = &errBuf
	if k.logger.IsVerbose() {
		fmt.Println(strings.Join(cmd.Args, " "))
		cmd.Stdout = os.Stdout
	}
	if err := cmd.Run(); err != nil {
		return false, errors.Wrapf(err, "disabling SELinux while installing kubelet so that containers can access the host filesystem failed with stderr %s", errBuf.String())
	}
	errBuf.Reset()

	cmd = exec.Command("sed", "-i", "s/^SELINUX=enforcing$/SELINUX=permissive/", "/etc/selinux/config")
	cmd.Stderr = &errBuf
	if k.logger.IsVerbose() {
		fmt.Println(strings.Join(cmd.Args, " "))
		cmd.Stdout = os.Stdout
	}
	if err := cmd.Run(); err != nil {
		return false, errors.Wrapf(err, "disabling SELinux while installing kubelet so that containers can access the host filesystem failed with stderr %s", errBuf.String())
	}
	errBuf.Reset()

	cmd = exec.Command("modprobe", "br_netfilter")
	cmd.Stderr = &errBuf
	if k.logger.IsVerbose() {
		fmt.Println(strings.Join(cmd.Args, " "))
		cmd.Stdout = os.Stdout
	}
	if err := cmd.Run(); err != nil {
		return false, errors.Wrapf(err, "loading module br_netfilter while installing kubelet failed with stderr %s", errBuf.String())
	}
	errBuf.Reset()

	file, err := os.Create("/etc/sysctl.d/k8s.conf")
	if err != nil {
		return false, errors.Wrap(err, "opening /etc/sysctl.d/k8s.conf in order to set net.bridge.bridge-nf-call-iptables to 1 while installing kubelet failed")
	}
	defer file.Close()

	file.Write(([]byte(`net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
		`)))
	if err != nil {
		return false, errors.Wrap(err, "writing to /etc/sysctl.d/k8s.conf in order to set net.bridge.bridge-nf-call-iptables to 1 while installing kubelet failed")
	}
	file.Close()

	cmd = exec.Command("sysctl", "--system")
	cmd.Stderr = &errBuf
	if k.logger.IsVerbose() {
		fmt.Println(strings.Join(cmd.Args, " "))
		cmd.Stdout = os.Stdout
	}

	if err := cmd.Run(); err != nil {
		return false, errors.Wrapf(err, "running sysctl --system in order to set net.bridge.bridge-nf-call-iptables to 1 while installing kubelet failed with stderr %s", errBuf.String())
	}
	errBuf.Reset()

	cmd = exec.Command("bash", "-c", "echo 1 > /proc/sys/net/ipv4/ip_forward")
	cmd.Stderr = &errBuf
	if k.logger.IsVerbose() {
		fmt.Println(strings.Join(cmd.Args, " "))
		cmd.Stdout = os.Stdout
	}

	return false, errors.Wrapf(cmd.Run(), "enabling ip forwarding by writing 1 to /proc/sys/net/ipv4/ip_forward failed with stderr %s", errBuf.String())
}
