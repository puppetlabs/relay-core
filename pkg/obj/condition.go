package obj

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConditionImage  = "relaysh/core:latest"
	ConditionScript = `#!/bin/bash
JQ="${JQ:-jq}"

CONDITIONS_URL="${CONDITIONS_URL:-conditions}"
VALUE_NAME="${VALUE_NAME:-success}"
POLLING_INTERVAL="${POLLING_INTERVAL:-5s}"
POLLING_ITERATIONS="${POLLING_ITERATIONS:-1080}"

for i in $(seq ${POLLING_ITERATIONS}); do
	CONDITIONS=$(curl "$METADATA_API_URL/${CONDITIONS_URL}")
	VALUE=$(echo $CONDITIONS | $JQ --arg value "$VALUE_NAME" -r '.[$value]')
	if [ -n "${VALUE}" ]; then
	if [ "$VALUE" = "true" ]; then
		exit 0
	fi
	if [ "$VALUE" = "false" ]; then
		exit 1
	fi
	fi
	sleep ${POLLING_INTERVAL}
done

exit 1
`
)

type Condition struct {
	Key    client.ObjectKey
	Object *tektonv1alpha1.Condition
}

var _ lifecycle.Loader = &Condition{}
var _ lifecycle.Ownable = &Condition{}
var _ lifecycle.Persister = &Condition{}

func (c *Condition) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(c.Object, owner)
}

func (c *Condition) Load(ctx context.Context, cl client.Client) (bool, error) {
	return helper.GetIgnoreNotFound(ctx, cl, c.Key, c.Object)
}

func (c *Condition) Persist(ctx context.Context, cl client.Client) error {
	return helper.CreateOrUpdate(ctx, cl, c.Object, helper.WithObjectKey(c.Key))
}

func NewCondition(key client.ObjectKey) *Condition {
	return &Condition{
		Key:    key,
		Object: &tektonv1alpha1.Condition{},
	}
}
