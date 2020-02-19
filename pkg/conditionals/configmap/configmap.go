package configmap

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ConditionalsManager struct {
	kubeclient kubernetes.Interface
	namespace  string
}

func (cm ConditionalsManager) GetByTaskID(ctx context.Context, taskID string) (string, errors.Error) {
    configMap, err := cm.kubeclient.CoreV1().ConfigMaps(cm.namespace).Get(taskID, metav1.GetOptions{})
    if err != nil {
        if kerrors.IsNotFound(err) {
            return "", errors.NewTaskConditionalsNotFoundForID(taskID).WithCause(err)
		}

		return "", errors.NewTaskConditionalsLookupError().WithCause(err)
    }

    if _, ok := configMap.Data["conditionals"]; !ok {
        return "", errors.NewTaskConditionalsNotFoundForID(taskID)
    }

    return configMap.Data["conditionals"], nil
}

func New(kc kubernetes.Interface, namespace string) *ConditionalsManager {
	return &ConditionalsManager{
		kubeclient: kc,
		namespace:  namespace,
	}
}
