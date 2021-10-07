package app

import (
	"context"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type APITriggerEventSink struct {
	Sink        *relayv1beta1.APITriggerEventSink
	TokenSecret *corev1obj.OpaqueSecret
}

var _ lifecycle.Loader = &APITriggerEventSink{}

func (tes *APITriggerEventSink) Load(ctx context.Context, cl client.Client) (bool, error) {
	return lifecycle.IgnoreNilLoader{tes.TokenSecret}.Load(ctx, cl)
}

func (tes *APITriggerEventSink) URL() string {
	return tes.Sink.URL
}

func (tes *APITriggerEventSink) Token() (string, bool) {
	if tes.Sink.Token != "" {
		return tes.Sink.Token, true
	} else if tes.TokenSecret != nil {
		return tes.TokenSecret.Data(tes.Sink.TokenFrom.SecretKeyRef.Key)
	}

	return "", false
}

func NewAPITriggerEventSink(namespace string, sink *relayv1beta1.APITriggerEventSink) *APITriggerEventSink {
	tes := &APITriggerEventSink{
		Sink: sink,
	}

	if sink.TokenFrom != nil && sink.TokenFrom.SecretKeyRef != nil {
		tes.TokenSecret = corev1obj.NewOpaqueSecret(client.ObjectKey{
			Namespace: namespace,
			Name:      sink.TokenFrom.SecretKeyRef.Name,
		})
	}

	return tes
}

type APIWorkflowExecutionSink struct {
	Sink        *relayv1beta1.APIWorkflowExecutionSink
	TokenSecret *corev1obj.OpaqueSecret
}

var _ lifecycle.Loader = &APIWorkflowExecutionSink{}

func (a *APIWorkflowExecutionSink) Load(ctx context.Context, cl client.Client) (bool, error) {
	return lifecycle.IgnoreNilLoader{a.TokenSecret}.Load(ctx, cl)
}

func (a *APIWorkflowExecutionSink) URL() string {
	return a.Sink.URL
}

func (a *APIWorkflowExecutionSink) Token() (string, bool) {
	if a.Sink.Token != "" {
		return a.Sink.Token, true
	} else if a.TokenSecret != nil {
		return a.TokenSecret.Data(a.Sink.TokenFrom.SecretKeyRef.Key)
	}

	return "", false
}

func NewAPIWorkflowExecutionSink(namespace string, sink *relayv1beta1.APIWorkflowExecutionSink) *APIWorkflowExecutionSink {
	a := &APIWorkflowExecutionSink{
		Sink: sink,
	}

	if sink.TokenFrom != nil && sink.TokenFrom.SecretKeyRef != nil {
		a.TokenSecret = corev1obj.NewOpaqueSecret(client.ObjectKey{
			Namespace: namespace,
			Name:      sink.TokenFrom.SecretKeyRef.Name,
		})
	}

	return a
}
