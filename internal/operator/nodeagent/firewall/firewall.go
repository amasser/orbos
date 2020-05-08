package firewall

import (
	"github.com/caos/orbos/internal/operator/nodeagent"
	"github.com/caos/orbos/internal/operator/nodeagent/dep"
	"github.com/caos/orbos/mntr"
)

func Ensurer(monitor mntr.Monitor, os dep.OperatingSystem, ignore []string) nodeagent.FirewallEnsurer {
	switch os {
	case dep.CentOS:
		return centosEnsurer(monitor, ignore)
	default:
		return noopEnsurer()
	}
}
