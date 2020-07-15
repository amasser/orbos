package mapping

import (
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes/resources"
	macherrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type CORS struct {
	Origins        string
	Methods        string
	Headers        string
	Credentials    bool
	ExposedHeaders string
	MaxAge         string
}

func AdaptFunc(name, namespace string, labels map[string]string, grpc bool, host, prefix, rewrite, service, timeoutMS, connectTimeoutMS string, cors *CORS) (resources.QueryFunc, resources.DestroyFunc, error) {
	group := "getambassador.io"
	version := "v2"
	kind := "Mapping"

	spec := map[string]interface{}{
		"host":    host,
		"rewrite": rewrite,
		"service": service,
	}
	if prefix != "" {
		spec["prefix"] = prefix
	}

	if timeoutMS != "" {
		spec["timeout_ms"] = timeoutMS
	}
	if connectTimeoutMS != "" {
		spec["connect_timeout_ms"] = connectTimeoutMS
	}
	if grpc {
		spec["grpc"] = grpc
	}

	if cors != nil {
		corsMap := map[string]interface{}{
			"origins":         cors.Origins,
			"methods":         cors.Methods,
			"headers":         cors.Headers,
			"credentials":     cors.Credentials,
			"exposed_headers": cors.ExposedHeaders,
			"max_age":         cors.MaxAge,
		}
		spec["cors"] = corsMap
	}

	crd := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       kind,
			"apiVersion": group + "/" + version,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels":    labels,
			},
			"spec": spec,
		}}

	return func(k8sClient *kubernetes.Client) (resources.EnsureFunc, error) {
			res, err := k8sClient.GetNamespacedCRDResource(group, version, kind, namespace, name)
			if err != nil && !macherrs.IsNotFound(err) {
				return nil, err
			}
			resourceVersion := ""
			if res != nil {
				meta := res.Object["metadata"].(map[string]interface{})
				resourceVersion = meta["resourceVersion"].(string)
			}

			if resourceVersion != "" {
				meta := crd.Object["metadata"].(map[string]interface{})
				meta["resourceVersion"] = resourceVersion
			}

			return func(k8sClient *kubernetes.Client) error {
				return k8sClient.ApplyNamespacedCRDResource(group, version, kind, namespace, name, crd)
			}, nil
		}, func(k8sClient *kubernetes.Client) error {
			//TODO
			return nil
		}, nil
}

/*apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: dev-accounts-v1
spec:
  host: accounts.zitadel.dev
  prefix: /
  rewrite: /login/
  service: http://ui-v1.dev-zitadel
  timeout_ms: 30000
  connect_timeout_ms: 30000*/
