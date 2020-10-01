package databases

import (
	"github.com/caos/orbos/internal/api"
	coredb "github.com/caos/orbos/internal/operator/database/kinds/databases/core"
	orbdb "github.com/caos/orbos/internal/operator/database/kinds/orb"
	"github.com/caos/orbos/mntr"
	"github.com/caos/orbos/pkg/git"
	"github.com/caos/orbos/pkg/kubernetes"
	"github.com/caos/orbos/pkg/tree"
)

func GetConnectionInfo(
	monitor mntr.Monitor,
	k8sClient *kubernetes.Client,
	gitClient *git.Client,
) (string, string, error) {
	desired, err := api.ReadDatabaseYml(gitClient)
	if err != nil {
		monitor.Error(err)
		return "", "", err
	}
	current := &tree.Tree{}

	query, _, err := orbdb.AdaptFunc("", "database")(monitor, desired, current)
	if err != nil {
		return "", "", err
	}

	queried := map[string]interface{}{}
	_, err = query(k8sClient, queried)
	if err != nil {
		return "", "", err
	}
	currentDB, err := coredb.ParseQueriedForDatabase(queried)
	if err != nil {
		return "", "", err
	}
	return currentDB.GetURL(), currentDB.GetPort(), nil
}
