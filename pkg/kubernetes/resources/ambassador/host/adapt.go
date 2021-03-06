package host

import (
	"github.com/caos/orbos/pkg/kubernetes"
	"github.com/caos/orbos/pkg/kubernetes/resources"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	group   = "getambassador.io"
	version = "v2"
	kind    = "Host"
)

func AdaptFuncToEnsure(namespace, name string, labels map[string]string, hostname string, authority string, privateKeySecret string, selector map[string]string, tlsSecret string) (resources.QueryFunc, error) {
	acme := map[string]interface{}{
		"authority": authority,
	}
	if privateKeySecret != "" {
		acme["privateKeySecret"] = map[string]interface{}{
			"name": privateKeySecret,
		}
	}

	selectorInternal := make(map[string]interface{}, 0)
	for k, v := range selector {
		selectorInternal[k] = v
	}
	crd := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       kind,
			"apiVersion": group + "/" + version,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels":    labels,
				"annotations": map[string]interface{}{
					"aes_res_changed": "true",
				},
			},
			"spec": map[string]interface{}{
				"hostname":     hostname,
				"acmeProvider": acme,
				"ambassador_id": []string{
					"default",
				},
				"selector": map[string]interface{}{
					"matchLabels": selectorInternal,
				},
				"tlsSecret": map[string]interface{}{
					"name": tlsSecret,
				},
			},
		}}

	return func(k8sClient kubernetes.ClientInt) (resources.EnsureFunc, error) {
		return func(k8sClient kubernetes.ClientInt) error {
			return k8sClient.ApplyNamespacedCRDResource(group, version, kind, namespace, name, crd)
		}, nil
	}, nil
}

func AdaptFuncToDestroy(namespace, name string) (resources.DestroyFunc, error) {
	return func(client kubernetes.ClientInt) error {
		return client.DeleteNamespacedCRDResource(group, version, kind, namespace, name)
	}, nil
}
