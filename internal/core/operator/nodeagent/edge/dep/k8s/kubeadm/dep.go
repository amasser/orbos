package kubeadm

import (
	"regexp"

	"github.com/caos/orbiter/internal/core/operator/common"
	"github.com/caos/orbiter/internal/core/operator/nodeagent"
	"github.com/caos/orbiter/internal/core/operator/nodeagent/edge/dep"
	"github.com/caos/orbiter/internal/core/operator/nodeagent/edge/dep/middleware"
	"github.com/caos/orbiter/internal/core/operator/nodeagent/edge/dep/k8s"
)

type Installer interface {
	isKubeadm()
	nodeagent.Installer
}

type kubeadmDep struct {
	manager    *dep.PackageManager
	os         dep.OperatingSystem
	normalizer *regexp.Regexp
	common     *k8s.Common
}

func New(os dep.OperatingSystem, manager *dep.PackageManager) Installer {
	return &kubeadmDep{manager, os, regexp.MustCompile(`\d+\.\d+\.\d+`), k8s.New(os, manager, "kubeadm")}
}

func (kubeadmDep) isKubeadm() {}

func (kubeadmDep) Is(other nodeagent.Installer) bool {
	_, ok := middleware.Unwrap(other).(Installer)
	return ok
}

func (k kubeadmDep) String() string { return "Kubeadm" }

func (*kubeadmDep) Equals(other nodeagent.Installer) bool {
	_, ok := other.(*kubeadmDep)
	return ok
}

func (k *kubeadmDep) Current() (common.Package, error) {
	return k.common.Current()
}

func (k *kubeadmDep) Ensure(remove common.Package, install common.Package) (bool, error) {
	return false, k.common.Ensure(remove, install)
}
