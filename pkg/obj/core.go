package obj

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Core struct {
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *v1alpha1.RelayCore
}

func (c *Core) Copy() *Core {
	return makeCore(c.Key, c.Object.DeepCopy())
}

func (c *Core) PersistStatus(ctx context.Context, cl client.Client) error {
	return cl.Status().Update(ctx, c.Object)
}

func makeCore(key client.ObjectKey, obj *v1alpha1.RelayCore) *Core {
	c := &Core{Key: key, Object: obj}
	c.NamespaceScopedAPIObject = helper.ForNamespaceScopedAPIObject(
		&c.Key,
		lifecycle.TypedObject{
			GVK:    v1alpha1.RelayCoreKind,
			Object: c.Object,
		},
	)

	return c
}

func NewCore(key client.ObjectKey) *Core {
	return makeCore(key, &v1alpha1.RelayCore{})
}

func NewCoreFromObject(obj *v1alpha1.RelayCore) *Core {
	return makeCore(client.ObjectKeyFromObject(obj), obj)
}
