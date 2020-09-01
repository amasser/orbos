package v1beta2

import (
	"github.com/caos/orbos/internal/operator/boom/api/v1beta2/resources"
	"github.com/caos/orbos/internal/operator/boom/api/v1beta2/storage"
	"github.com/caos/orbos/internal/operator/boom/api/v1beta2/toleration"
)

type LogCollection struct {
	//Flag if tool should be deployed
	//@default: false
	Deploy bool `json:"deploy" yaml:"deploy"`
	//Spec to define how the persistence should be handled
	FluentdPVC *storage.Spec `json:"fluentdStorage,omitempty" yaml:"fluentdStorage,omitempty"`
	//NodeSelector for deployment
	NodeSelector map[string]string `json:"nodeSelector,omitempty" yaml:"nodeSelector,omitempty"`
	//Tolerations to run fluentbit on nodes
	Tolerations toleration.Tolerations `json:"tolerations,omitempty" yaml:"tolerations,omitempty"`
	//Resource requirements
	Resources *resources.Resources `json:"resources,omitempty" yaml:"resources,omitempty"`
}
