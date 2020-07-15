package host

import (
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes/resources"
	macherrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func AdaptFunc(name, namespace string, labels map[string]string, hostname string, authority string, privateKeySecret string, selector map[string]string, tlsSecret string) (resources.QueryFunc, resources.DestroyFunc, error) {
	group := "getambassador.io"
	version := "v2"
	kind := "Host"

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
				"ambassadorId": []string{
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
