package providers

import (
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/core/infra"
	"regexp"
	"strings"

	"github.com/caos/orbos/internal/operator/orbiter"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/providers/gce"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/providers/static"
	"github.com/caos/orbos/internal/secret"
	"github.com/caos/orbos/internal/tree"
	"github.com/caos/orbos/mntr"
	"github.com/pkg/errors"
)

var alphanum = regexp.MustCompile("[^a-zA-Z0-9]+")

func GetQueryAndDestroyFuncs(
	monitor mntr.Monitor,
	provID string,
	providerTree *tree.Tree,
	providerCurrent *tree.Tree,
	whitelistChan chan []*orbiter.CIDR,
	finishedChan chan bool,
	orbiterCommit, repoURL, repoKey string,
) (
	orbiter.QueryFunc,
	orbiter.DestroyFunc,
	bool,
	error,
) {

	monitor = monitor.WithFields(map[string]interface{}{"provider": provID})

	wlFunc := func() []*orbiter.CIDR {
		monitor.Debug("Reading whitelist")
		return <-whitelistChan
	}

	switch providerTree.Common.Kind {
	case "orbiter.caos.ch/GCEProvider":
		return gce.AdaptFunc(
			provID,
			alphanum.ReplaceAllString(strings.TrimSuffix(strings.TrimPrefix(repoURL, "git@"), ".git"), "-"),
			wlFunc,
			orbiterCommit, repoURL, repoKey,
		)(
			monitor,
			finishedChan,
			providerTree,
			providerCurrent,
		)
	case "orbiter.caos.ch/StaticProvider":

		adaptFunc := func() (orbiter.QueryFunc, orbiter.DestroyFunc, bool, error) {
			return static.AdaptFunc(
				provID,
				func() []*orbiter.CIDR {
					monitor.Debug("Reading whitelist")
					return <-whitelistChan
				},
				orbiterCommit, repoURL, repoKey,
			)(
				monitor.WithFields(map[string]interface{}{"provider": provID}),
				finishedChan,
				providerTree,
				providerCurrent)
		}
		return orbiter.AdaptFuncGoroutine(adaptFunc)
	default:
		return nil, nil, false, errors.Errorf("unknown provider kind %s", providerTree.Common.Kind)
	}
}

func GetSecrets(
	monitor mntr.Monitor,
	providerTree *tree.Tree,
) (
	map[string]*secret.Secret,
	error,
) {
	switch providerTree.Common.Kind {
	case "orbiter.caos.ch/GCEProvider":
		return gce.SecretsFunc()(
			monitor,
			providerTree,
		)
	case "orbiter.caos.ch/StaticProvider":
		return static.SecretsFunc()(
			monitor,
			providerTree,
		)
	default:
		return nil, errors.Errorf("unknown provider kind %s", providerTree.Common.Kind)
	}
}

func RewriteMasterkey(
	monitor mntr.Monitor,
	newMasterkey string,
	providerTree *tree.Tree,
) (
	map[string]*secret.Secret,
	error,
) {
	switch providerTree.Common.Kind {
	case "orbiter.caos.ch/GCEProvider":
		return gce.RewriteFunc(
			newMasterkey,
		)(
			monitor,
			providerTree,
		)
	case "orbiter.caos.ch/StaticProvider":
		return static.RewriteFunc(
			newMasterkey,
		)(
			monitor,
			providerTree,
		)
	default:
		return nil, errors.Errorf("unknown provider kind %s", providerTree.Common.Kind)
	}
}

func ListMachines(
	monitor mntr.Monitor,
	providerTree *tree.Tree,
) (
	map[string]infra.Machine,
	error,
) {
	switch providerTree.Common.Kind {
	case "orbiter.caos.ch/GCEProvider":
		return nil, nil
	case "orbiter.caos.ch/StaticProvider":
		return static.ListMachines(monitor, providerTree)
	default:
		return nil, errors.Errorf("unknown provider kind %s", providerTree.Common.Kind)
	}
}
