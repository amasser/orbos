//go:generate goderive -autoname -dedup .

package operator

import (
	"sync"

	"github.com/mitchellh/mapstructure"

	"github.com/caos/orbiter/internal/core/logging"
)

var nodeagentBytesGZIPBase64 string

type NodeAgentCurrent struct {
	NodeIsReady bool `mapstructure:"ready" yaml:"ready"`
	Software    Software
	Open        Firewall
	Version     string
	changer     *changer
}

type Software struct {
	Swap             Package `yaml:",omitempty"`
	Kubelet          Package `yaml:",omitempty"`
	Kubeadm          Package `yaml:",omitempty"`
	Kubectl          Package `yaml:",omitempty"`
	Containerruntime Package `yaml:",omitempty"`
	KeepaliveD       Package `yaml:",omitempty"`
	Nginx            Package `yaml:",omitempty"`
	Hostname         Package `yaml:",omitempty"`
}

type Package struct {
	Version string            `yaml:",omitempty"`
	Config  map[string]string `yaml:",omitempty"`
}

func (p *Package) Equals(other *Package) bool {
	return deriveEqualPkg(p, other)
}

func (s *Software) Equals(other *Software) bool {
	return deriveEqualSoftware(s, other)
}

type Firewall map[string]Allowed

func (f Firewall) Equals(other Firewall) bool {
	return deriveEqualFirewall(f, other)
}

type Allowed struct {
	Port     string
	Protocol string
}

type NodeAgentSpec struct {
	ChangesAllowed bool
	//	RebootEnabled  bool
	Software *Software
	Firewall Firewall
}

func (n *NodeAgentCurrent) AllowChanges() {
	n.changer.desire(func(spec *NodeAgentSpec) {
		spec.ChangesAllowed = true
	})
}

func (n *NodeAgentCurrent) DesireFirewall(fw Firewall) {
	n.changer.desire(func(spec *NodeAgentSpec) {
		spec.Firewall = fw
	})
}

func (n *NodeAgentCurrent) DesireSoftware(sw *Software) {
	n.changer.desire(func(spec *NodeAgentSpec) {
		if spec.Software == nil && sw != nil {
			spec.Software = &Software{}
		}

		zeroPkg := &Package{}

		if !sw.Containerruntime.Equals(zeroPkg) {
			spec.Software.Containerruntime = sw.Containerruntime
		}

		if !sw.KeepaliveD.Equals(zeroPkg) {
			spec.Software.KeepaliveD = sw.KeepaliveD
		}

		if !sw.Nginx.Equals(zeroPkg) {
			spec.Software.Nginx = sw.Nginx
		}

		if !sw.Kubeadm.Equals(zeroPkg) {
			spec.Software.Kubeadm = sw.Kubeadm
		}

		if !sw.Kubelet.Equals(zeroPkg) {
			spec.Software.Kubelet = sw.Kubelet
		}

		if !sw.Kubectl.Equals(zeroPkg) {
			spec.Software.Kubectl = sw.Kubectl
		}

		if !sw.Swap.Equals(zeroPkg) {
			spec.Software.Swap = sw.Swap
		}

		if !sw.Hostname.Equals(zeroPkg) {
			spec.Software.Hostname = sw.Hostname
		}
	})
}

type changer struct {
	path    []string
	kind    map[string]interface{}
	changes chan<- *nodeAgentChange
}

func (c *changer) desire(mutate func(*NodeAgentSpec)) {
	newSpec := &NodeAgentSpec{}
	mapstructure.Decode(c.kind["spec"], newSpec)
	mutate(newSpec)
	c.kind["spec"] = newSpec
	c.changes <- &nodeAgentChange{
		path: c.path,
		spec: newSpec,
	}
}

type NodeAgentKind struct {
	Kind    string
	Version string
	Spec    interface{}
	Current *CurrentNodeAgentKind `yaml:",omitempty"`
}

type CurrentNodeAgentKind struct {
	Version string
	State   *NodeAgentCurrent
}

type muxMap struct {
	mux  sync.Mutex
	data map[string]interface{}
}

func newNodeAgentCurrent(logger logging.Logger, path []string, nodeAgentSource map[string]interface{}, changes chan<- *nodeAgentChange) *NodeAgentCurrent {

	naKind, err := drillIn(logger.WithFields(map[string]interface{}{
		"purpose": "find node agent",
		"config":  "current",
	}), nodeAgentSource, append([]string{"current", "state"}, path...), true)
	if err != nil {
		panic(err)
	}

	changer := &changer{path, naKind, changes}
	fallbackCurrent := &NodeAgentCurrent{changer: changer}

	kind := &NodeAgentKind{}
	mapstructure.Decode(naKind, kind)
	if kind.Current == nil {
		return fallbackCurrent
	}

	switch kind.Current.Version {
	case "v1":
		kind.Current.State.changer = changer
		return kind.Current.State
	default:
		return fallbackCurrent
	}
}
