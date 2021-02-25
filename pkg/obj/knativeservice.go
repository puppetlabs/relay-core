package obj

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	KnativeServiceKind = servingv1.SchemeGroupVersion.WithKind("Service")
)

type KnativeService struct {
	Key    client.ObjectKey
	Object *servingv1.Service
}

var _ lifecycle.LabelAnnotatableFrom = &KnativeService{}
var _ lifecycle.Loader = &KnativeService{}
var _ lifecycle.Ownable = &KnativeService{}
var _ lifecycle.Persister = &KnativeService{}

func (ks *KnativeService) LabelAnnotateFrom(ctx context.Context, from metav1.Object) {
	helper.CopyLabelsAndAnnotations(&ks.Object.ObjectMeta, from)
}

func (ks *KnativeService) Load(ctx context.Context, cl client.Client) (bool, error) {
	return helper.GetIgnoreNotFound(ctx, cl, ks.Key, ks.Object)
}

func (ks *KnativeService) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(ks.Object, owner)
}

func (ks *KnativeService) Persist(ctx context.Context, cl client.Client) error {
	return helper.CreateOrUpdate(ctx, cl, ks.Object, helper.WithObjectKey(ks.Key))
}

func NewKnativeService(key client.ObjectKey) *KnativeService {
	return &KnativeService{
		Key:    key,
		Object: &servingv1.Service{},
	}
}

type KnativeServiceResult struct {
	KnativeService *KnativeService
	Error          error
}

func AsKnativeServiceResult(ks *KnativeService, err error) *KnativeServiceResult {
	return &KnativeServiceResult{
		KnativeService: ks,
		Error:          err,
	}
}
