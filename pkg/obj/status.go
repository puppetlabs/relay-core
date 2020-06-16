package obj

import (
	"time"

	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func UpdateStatusConditionIfTransitioned(target *relayv1beta1.Condition, fn func() relayv1beta1.Condition) {
	if target == nil {
		return
	}

	nc := fn()

	if nc.Status == target.Status && nc.Reason == target.Reason && nc.Message == target.Message {
		return
	}

	if nc.LastTransitionTime.Time.IsZero() {
		nc.LastTransitionTime = metav1.Time{Time: time.Now()}
	}

	*target = nc
}

func AggregateStatusConditions(conds ...relayv1beta1.Condition) corev1.ConditionStatus {
	if len(conds) == 0 {
		return corev1.ConditionUnknown
	}

	all := corev1.ConditionTrue

	for _, cond := range conds {
		switch cond.Status {
		case corev1.ConditionTrue:
		case corev1.ConditionFalse:
			return corev1.ConditionFalse
		case corev1.ConditionUnknown:
			all = corev1.ConditionUnknown
		}
	}

	return all
}
