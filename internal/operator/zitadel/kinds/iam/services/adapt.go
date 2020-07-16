package services

import (
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes/resources/service"
	"github.com/caos/orbos/internal/operator/zitadel"
)

func AdaptFunc(
	namespace string,
	labels map[string]string,
	grpcServiceName string,
	grpcPort int,
	httpServiceName string,
	httpPort int,
	uiServiceName string,
	uiPort int,
) (
	zitadel.QueryFunc,
	zitadel.DestroyFunc,
	error,
) {
	destroyGRPC, err := service.AdaptFuncToDestroy(grpcServiceName, namespace)
	if err != nil {
		return nil, nil, err
	}

	destroyHTTP, err := service.AdaptFuncToDestroy(httpServiceName, namespace)
	if err != nil {
		return nil, nil, err
	}

	destroyUI, err := service.AdaptFuncToDestroy(uiServiceName, namespace)
	if err != nil {
		return nil, nil, err
	}

	destroyers := []zitadel.DestroyFunc{
		zitadel.ResourceDestroyToZitadelDestroy(destroyGRPC),
		zitadel.ResourceDestroyToZitadelDestroy(destroyHTTP),
		zitadel.ResourceDestroyToZitadelDestroy(destroyUI),
	}

	grpcPorts := []service.Port{
		{Name: "grpc", Port: grpcPort, TargetPort: "grpc"},
	}
	queryGRPC, err := service.AdaptFuncToEnsure(grpcServiceName, namespace, labels, grpcPorts, "", labels, false, "", "")
	if err != nil {
		return nil, nil, err
	}

	httpPorts := []service.Port{
		{Name: "http", Port: httpPort, TargetPort: "http"},
	}
	queryHTTP, err := service.AdaptFuncToEnsure(httpServiceName, namespace, labels, httpPorts, "", labels, false, "", "")
	if err != nil {
		return nil, nil, err
	}

	uiPorts := []service.Port{
		{Name: "ui", Port: uiPort, TargetPort: "ui"},
	}
	queryUI, err := service.AdaptFuncToEnsure(uiServiceName, namespace, labels, uiPorts, "", labels, false, "", "")
	if err != nil {
		return nil, nil, err
	}

	queriers := []zitadel.QueryFunc{
		zitadel.ResourceQueryToZitadelQuery(queryGRPC),
		zitadel.ResourceQueryToZitadelQuery(queryHTTP),
		zitadel.ResourceQueryToZitadelQuery(queryUI),
	}

	return func(k8sClient *kubernetes.Client, queried map[string]interface{}) (zitadel.EnsureFunc, error) {
			return zitadel.QueriersToEnsureFunc(queriers, k8sClient, queried)
		},
		zitadel.DestroyersToDestroyFunc(destroyers),
		nil
}
