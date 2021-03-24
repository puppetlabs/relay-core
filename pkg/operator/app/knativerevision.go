package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/obj"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KnativeRevisionSet struct {
	ListOptions *client.ListOptions

	Revisions []*obj.KnativeRevision
}

var _ lifecycle.Loader = &KnativeRevisionSet{}

func (krs *KnativeRevisionSet) Load(ctx context.Context, cl client.Client) (bool, error) {
	revs := &servingv1.RevisionList{}
	if err := cl.List(ctx, revs, krs.ListOptions); err != nil {
		return false, err
	}

	krs.Revisions = make([]*obj.KnativeRevision, len(revs.Items))
	for i := range revs.Items {
		krs.Revisions[i] = obj.NewKnativeRevisionFromObject(&revs.Items[i])
	}

	return true, nil
}

func NewKnativeRevisionSet(opts ...client.ListOption) *KnativeRevisionSet {
	o := &client.ListOptions{}
	o.ApplyOptions(opts)

	return &KnativeRevisionSet{
		ListOptions: o,
	}
}

func NewKnativeRevisionSetForKnativeService(ks *obj.KnativeService) *KnativeRevisionSet {
	return NewKnativeRevisionSet(
		client.InNamespace(ks.Key.Namespace),
		client.MatchingLabels{
			"serving.knative.dev/service": ks.Key.Name,
		},
	)
}
