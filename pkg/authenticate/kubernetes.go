package authenticate

import (
	"context"
	"fmt"
	"net"

	tekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	KubernetesTokenAnnotation   = "relay.sh/token"
	KubernetesSubjectAnnotation = "relay.sh/subject"
)

type KubernetesIntermediaryMetadata struct {
	NamespaceUID types.UID
}

type KubernetesChainIntermediaryFunc func(ctx context.Context, raw Raw, md *KubernetesIntermediaryMetadata) (Intermediary, error)

type TektonInterface = tekton.Interface

type KubernetesInterface struct {
	kubernetes.Interface
	TektonInterface
}

func NewKubernetesInterfaceForConfig(cfg *rest.Config) (*KubernetesInterface, error) {
	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	tkc, err := tekton.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &KubernetesInterface{
		Interface:       kc,
		TektonInterface: tkc,
	}, nil
}

// KubernetesIntermediary looks up a pod by IP and reads the value of an
// annotation as the authentication credential.
type KubernetesIntermediary struct {
	client *KubernetesInterface
	ip     net.IP
}

var _ Intermediary = &KubernetesIntermediary{}

func (ki *KubernetesIntermediary) next(ctx context.Context, state *Authentication) (Raw, *KubernetesIntermediaryMetadata, error) {
	if len(ki.ip) == 0 || ki.ip.IsUnspecified() {
		return nil, nil, ErrNotFound
	}

	pods, err := ki.client.CoreV1().Pods("").List(metav1.ListOptions{
		FieldSelector: fmt.Sprintf("status.podIP=%s", ki.ip),
	})
	if err != nil {
		return nil, nil, err
	}

	switch len(pods.Items) {
	case 0:
		// Perhaps not a valid source IP.
		// TODO: Security incident?
		return nil, nil, ErrNotFound
	case 1:
		// Only acceptable case.
	default:
		// Multiple pods with the same IP? This is just nonsense and we'll throw
		// out the request.
		return nil, nil, ErrNotFound
	}

	pod := pods.Items[0]

	ns, err := ki.client.CoreV1().Namespaces().Get(pod.GetNamespace(), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil, nil, ErrNotFound
	} else if err != nil {
		return nil, nil, err
	}

	subject := pod.GetAnnotations()[KubernetesSubjectAnnotation]

	tok, found := pod.GetAnnotations()[KubernetesTokenAnnotation]
	if !found || tok == "" {
		// Right now we don't propagate annotations from TaskRuns to condition
		// pods. Let's check the owner and try to pull it for its annotation.
		//
		// TODO: Implement this in Tekton and remove this entire block.
		owner := metav1.GetControllerOf(&pod)
		if owner == nil {
			return nil, nil, ErrNotFound
		}

		gvk := schema.FromAPIVersionAndKind(owner.APIVersion, owner.Kind)
		if gvk.Group != "tekton.dev" || gvk.Kind != "TaskRun" {
			return nil, nil, ErrNotFound
		}

		tr, err := ki.client.TektonV1beta1().TaskRuns(ns.GetName()).Get(owner.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil, nil, ErrNotFound
		} else if err != nil {
			return nil, nil, err
		}

		// Check and copy out the annotations.
		subject = tr.GetAnnotations()[KubernetesSubjectAnnotation]

		tok, found = tr.GetAnnotations()[KubernetesTokenAnnotation]
		if !found || tok == "" {
			return nil, nil, ErrNotFound
		}
	}

	md := &KubernetesIntermediaryMetadata{
		NamespaceUID: ns.GetUID(),
	}

	// Namespace validation.
	state.AddValidator(ValidatorFunc(func(ctx context.Context, claims *Claims) (bool, error) {
		if claims.KubernetesNamespaceUID == "" {
			return false, nil
		}

		return md.NamespaceUID == types.UID(claims.KubernetesNamespaceUID), nil
	}))

	state.AddValidator(ValidatorFunc(func(ctx context.Context, claims *Claims) (bool, error) {
		if claims.Subject == "" {
			return false, nil
		}

		return subject == claims.Subject, nil
	}))

	return Raw(tok), md, nil
}

func (ki *KubernetesIntermediary) Chain(fn KubernetesChainIntermediaryFunc) Intermediary {
	return IntermediaryFunc(func(ctx context.Context, state *Authentication) (Raw, error) {
		raw, md, err := ki.next(ctx, state)
		if err != nil {
			return nil, err
		}

		next, err := fn(ctx, raw, md)
		if err != nil {
			return nil, err
		}

		return next.Next(ctx, state)
	})
}

func (ki *KubernetesIntermediary) Next(ctx context.Context, state *Authentication) (Raw, error) {
	raw, _, err := ki.next(ctx, state)
	return raw, err
}

func NewKubernetesIntermediary(client *KubernetesInterface, ip net.IP) *KubernetesIntermediary {
	return &KubernetesIntermediary{
		client: client,
		ip:     ip,
	}
}
