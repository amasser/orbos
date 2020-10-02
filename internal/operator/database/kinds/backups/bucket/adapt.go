package bucket

import (
	"github.com/caos/orbos/internal/operator/core"
	"github.com/caos/orbos/internal/operator/database/kinds/backups/bucket/backup"
	"github.com/caos/orbos/internal/operator/database/kinds/backups/bucket/clean"
	"github.com/caos/orbos/internal/operator/database/kinds/backups/bucket/restore"
	coreDB "github.com/caos/orbos/internal/operator/database/kinds/databases/core"
	"github.com/caos/orbos/mntr"
	"github.com/caos/orbos/pkg/kubernetes"
	"github.com/caos/orbos/pkg/kubernetes/resources/secret"
	"github.com/caos/orbos/pkg/tree"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func AdaptFunc(
	name string,
	namespace string,
	labels map[string]string,
	checkDBReady core.EnsureFunc,
	timestamp string,
	nodeselector map[string]string,
	tolerations []corev1.Toleration,
	version string,
	features []string,
) core.AdaptFunc {
	return func(monitor mntr.Monitor, desired *tree.Tree, current *tree.Tree) (queryFunc core.QueryFunc, destroyFunc core.DestroyFunc, err error) {
		secretName := "backup-serviceaccountjson"
		secretKey := "serviceaccountjson"

		internalMonitor := monitor.WithField("component", "backup")

		desiredKind, err := parseDesiredV0(desired)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parsing desired state failed")
		}
		desired.Parsed = desiredKind

		if !monitor.IsVerbose() && desiredKind.Spec.Verbose {
			internalMonitor.Verbose()
		}

		//queryM, destroyM, checkMigrationDone, cleanupMigration, err := migration.AdaptFunc(monitor, namespace, "restore", labels, secretPasswordName, migrationUser, users, nodeselector, tolerations)

		destroyS, err := secret.AdaptFuncToDestroy(namespace, secretName)
		if err != nil {
			return nil, nil, err
		}

		queryS, err := secret.AdaptFuncToEnsure(namespace, secretName, labels, map[string]string{secretKey: desiredKind.Spec.ServiceAccountJSON.Value})
		if err != nil {
			return nil, nil, err
		}

		_, destroyB, err := backup.AdaptFunc(
			internalMonitor,
			name,
			namespace,
			labels,
			[]string{},
			checkDBReady,
			desiredKind.Spec.Bucket,
			desiredKind.Spec.Cron,
			secretName,
			secretKey,
			timestamp,
			nodeselector,
			tolerations,
			features,
			version,
		)

		_, destroyR, _, err := restore.ApplyFunc(
			monitor,
			name,
			namespace,
			labels,
			[]string{},
			desiredKind.Spec.Bucket,
			timestamp,
			nodeselector,
			tolerations,
			checkDBReady,
			secretName,
			secretKey,
			version,
		)

		_, destroyC, _, err := clean.ApplyFunc(
			monitor,
			name,
			namespace,
			labels,
			[]string{},
			nodeselector,
			tolerations,
			checkDBReady,
			secretName,
			secretKey,
			version,
		)
		destroyers := make([]core.DestroyFunc, 0)
		for _, feature := range features {
			switch feature {
			case "backup", "instantbackup":
				destroyers = append(destroyers,
					core.ResourceDestroyToZitadelDestroy(destroyS),
					destroyB,
				)
			case "clear":
				destroyers = append(destroyers,
					destroyC,
				)
			case "restore":
				destroyers = append(destroyers,
					destroyR,
				)
			}
		}

		return func(k8sClient *kubernetes.Client, queried map[string]interface{}) (core.EnsureFunc, error) {
				currentDB, err := coreDB.ParseQueriedForDatabase(queried)
				if err != nil {
					return nil, err
				}

				databases, err := currentDB.GetListDatabasesFunc()(k8sClient)
				if err != nil {
					databases = []string{}
				}

				queryB, _, err := backup.AdaptFunc(
					internalMonitor,
					name,
					namespace,
					labels,
					databases,
					checkDBReady,
					desiredKind.Spec.Bucket,
					desiredKind.Spec.Cron,
					secretName,
					secretKey,
					timestamp,
					nodeselector,
					tolerations,
					features,
					version,
				)
				if err != nil {
					return nil, err
				}

				queryR, _, checkAndCleanupR, err := restore.ApplyFunc(
					monitor,
					name,
					namespace,
					labels,
					databases,
					desiredKind.Spec.Bucket,
					timestamp,
					nodeselector,
					tolerations,
					checkDBReady,
					secretName,
					secretKey,
					version,
				)
				if err != nil {
					return nil, err
				}

				queryC, _, checkAndCleanupC, err := clean.ApplyFunc(
					monitor,
					name,
					namespace,
					labels,
					databases,
					nodeselector,
					tolerations,
					checkDBReady,
					secretName,
					secretKey,
					version,
				)
				if err != nil {
					return nil, err
				}

				queriers := make([]core.QueryFunc, 0)
				if databases != nil && len(databases) != 0 {
					for _, feature := range features {
						switch feature {
						case "backup", "instantbackup":
							queriers = append(queriers,
								core.ResourceQueryToZitadelQuery(queryS),
								queryB,
							)
						case "clear":
							queriers = append(queriers,
								queryC,
								core.EnsureFuncToQueryFunc(checkAndCleanupC),
							)
						case "restore":
							queriers = append(queriers,
								queryR,
								core.EnsureFuncToQueryFunc(checkAndCleanupR),
							)
						}
					}
				}

				return core.QueriersToEnsureFunc(internalMonitor, false, queriers, k8sClient, queried)
			},
			core.DestroyersToDestroyFunc(internalMonitor, destroyers),
			nil
	}
}
