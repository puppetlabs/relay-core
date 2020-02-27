package configmap

import (
	"context"

	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/parse"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ConditionalsManager struct {
	kubeclient kubernetes.Interface
	namespace  string
}

func (cm ConditionalsManager) Get(ctx context.Context, taskHash task.Hash) (parse.Tree, errors.Error) {
	taskHashKey := taskHash.HexEncoding()

	configMap, err := cm.kubeclient.CoreV1().ConfigMaps(cm.namespace).Get(taskHashKey, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.NewTaskConditionalsNotFoundForID(taskHashKey).WithCause(err)
		}

		return nil, errors.NewTaskConditionalsLookupError().WithCause(err)
	}

	if _, ok := configMap.Data["conditionals"]; !ok {
		return nil, errors.NewTaskConditionalsNotFoundForID(taskHashKey)
	}

	tree, perr := parse.ParseJSONString(configMap.Data["conditionals"])
	if perr != nil {
		return nil, errors.NewTaskConditionalsDecodingError().WithCause(perr)
	}

	return tree, nil
}

func New(kc kubernetes.Interface, namespace string) *ConditionalsManager {
	return &ConditionalsManager{
		kubeclient: kc,
		namespace:  namespace,
	}
}
