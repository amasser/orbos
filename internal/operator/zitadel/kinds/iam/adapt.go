package iam

import (
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes/resources"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes/resources/service"
	"github.com/caos/orbos/internal/operator/zitadel"
	"github.com/caos/orbos/internal/operator/zitadel/kinds/databases"
	"github.com/caos/orbos/internal/operator/zitadel/kinds/iam/deployment"
	"github.com/caos/orbos/internal/operator/zitadel/kinds/iam/migration"
	"github.com/caos/orbos/internal/tree"
	"github.com/caos/orbos/mntr"
	"github.com/pkg/errors"
)

func AdaptFunc() zitadel.AdaptFunc {
	return func(
		monitor mntr.Monitor,
		desired *tree.Tree,
		current *tree.Tree,
	) (
		zitadel.QueryFunc,
		zitadel.DestroyFunc,
		error,
	) {

		queriers := make([]resources.QueryFunc, 0)
		destroyers := make([]resources.DestroyFunc, 0)

		desiredKind, err := parseDesiredV0(desired)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parsing desired state failed")
		}
		desired.Parsed = desiredKind

		databaseCurrent := &tree.Tree{}
		queryDB, destroyDB, err := databases.GetQueryAndDestroyFuncs(monitor, desiredKind.Database, databaseCurrent)

		namespaceStr := "caos-zitadel"
		labels := map[string]string{"app.kubernetes.io/managed-by": "zitadel.caos.ch"}

		queryD, destroyD, err := deployment.AdaptFunc(namespaceStr, labels, desiredKind.Spec.ReplicaCount)
		if err != nil {
			return nil, nil, err
		}

		accountsPorts := []service.Port{
			{Name: "http", Port: 80, TargetPort: "accounts-http"},
		}
		queryS, destroyS, err := service.AdaptFunc("accounts-v1", namespaceStr, labels, accountsPorts, "", labels, false, "", "")
		if err != nil {
			return nil, nil, err
		}

		serviceApiAdminPorts := []service.Port{
			{Name: "rest", Port: 80, TargetPort: "admin-rest"},
			{Name: "grpc", Port: 8080, TargetPort: "admin-grpc"},
		}
		querySAA, destroySAA, err := service.AdaptFunc("api-admin-v1", namespaceStr, labels, serviceApiAdminPorts, "", labels, false, "", "")
		if err != nil {
			return nil, nil, err
		}

		serviceApiAuthPorts := []service.Port{
			{Name: "rest", Port: 80, TargetPort: "auth-rest"},
			{Name: "issuer", Port: 7070, TargetPort: "issuer-rest"},
			{Name: "grpc", Port: 8080, TargetPort: "auth-grpc"},
		}
		queryAA, destroyAA, err := service.AdaptFunc("api-auth-v1", namespaceStr, labels, serviceApiAuthPorts, "", labels, false, "", "")
		if err != nil {
			return nil, nil, err
		}

		serviceApiMgmtPorts := []service.Port{
			{Name: "rest", Port: 80, TargetPort: "management-rest"},
			{Name: "grpc", Port: 8080, TargetPort: "management-grpc"},
		}
		querySAM, destroySAM, err := service.AdaptFunc("api-management-v1", namespaceStr, labels, serviceApiMgmtPorts, "", labels, false, "", "")
		if err != nil {
			return nil, nil, err
		}

		serviceConsolePorts := []service.Port{
			{Name: "http", Port: 80, TargetPort: "console-http"},
		}
		querySC, destroySC, err := service.AdaptFunc("console-v1", namespaceStr, labels, serviceConsolePorts, "", labels, false, "", "")
		if err != nil {
			return nil, nil, err
		}

		queryM, destroyM, err := migration.AdaptFunc(namespaceStr, labels, "migrate-db")
		if err != nil {
			return nil, nil, err
		}

		queriers = append(queriers, queryD, queryS, querySAA, queryAA, querySAM, querySC, queryM)
		destroyers = append(destroyers, destroyD, destroyS, destroySAA, destroyAA, destroySAM, destroySC, destroyM)

		return func(k8sClient *kubernetes.Client) (zitadel.EnsureFunc, error) {
				ensureDB, err := queryDB(k8sClient)
				if err != nil {
					return nil, err
				}

				ensurers := make([]resources.EnsureFunc, 0)
				for _, querier := range queriers {
					ensurer, err := querier()
					if err != nil {
						return nil, err
					}
					ensurers = append(ensurers, ensurer)
				}

				return func(k8sClient *kubernetes.Client) error {
					if err := ensureDB(k8sClient); err != nil {
						return err
					}

					for _, ensurer := range ensurers {
						if err := ensurer(k8sClient); err != nil {
							return err
						}
					}
					return nil
				}, nil
			}, func(k8sClient *kubernetes.Client) error {
				if err := destroyDB(k8sClient); err != nil {
					return err
				}
				for _, destroyer := range destroyers {
					if err := destroyer(k8sClient); err != nil {
						return err
					}
				}
				return nil
			},
			nil
	}

}
