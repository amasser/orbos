package clusters

import (
	"github.com/caos/orbiter/internal/operator/orbiter"
	"github.com/caos/orbiter/internal/operator/orbiter/kinds/clusters/kubernetes"
	"github.com/caos/orbiter/internal/orb"
	"github.com/caos/orbiter/internal/secret"
	"github.com/caos/orbiter/internal/tree"
	"github.com/caos/orbiter/mntr"
	"github.com/pkg/errors"
)

func GetQueryAndDestroyFuncs(
	monitor mntr.Monitor,
	orb *orb.Orb,
	clusterID string,
	clusterTree *tree.Tree,
	orbiterCommit string,
	oneoff bool,
	deployOrbiterAndBoom bool,
	clusterCurrent *tree.Tree,
	destroyProviders func() (map[string]interface{}, error),
) (
	orbiter.QueryFunc,
	orbiter.DestroyFunc,
	bool,
	error,
) {

	switch clusterTree.Common.Kind {
	case "orbiter.caos.ch/KubernetesCluster":
		return kubernetes.AdaptFunc(
			orb,
			orbiterCommit,
			clusterID,
			oneoff,
			deployOrbiterAndBoom,
			destroyProviders,
		)(
			monitor.WithFields(map[string]interface{}{"cluster": clusterID}),
			clusterTree,
			clusterCurrent,
		)
		//				subassemblers[provIdx] = static.New(providerPath, generalOverwriteSpec, staticadapter.New(providermonitor, providerID, "/healthz", updatesDisabled, cfg.NodeAgent))
	default:
		return nil, nil, false, errors.Errorf("unknown cluster kind %s", clusterTree.Common.Kind)
	}
}

func GetSecrets(
	monitor mntr.Monitor,
	orb *orb.Orb,
	clusterTree *tree.Tree,
) (
	map[string]*secret.Secret,
	error,
) {

	switch clusterTree.Common.Kind {
	case "orbiter.caos.ch/KubernetesCluster":
		return kubernetes.SecretFunc(orb)(
			monitor,
			clusterTree,
		)
	default:
		return nil, errors.Errorf("unknown cluster kind %s", clusterTree.Common.Kind)
	}
}