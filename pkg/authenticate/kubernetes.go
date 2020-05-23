package authenticate

import (
	"context"
	"fmt"
	"net"

	tekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
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
		return nil, nil, &NotFoundError{Reason: "kubernetes: no IP address to look up"}
	}

	pods, err := ki.client.CoreV1().Pods("").List(metav1.ListOptions{
		FieldSelector: fields.Set{
			"status.podIP": ki.ip.String(),
			"status.phase": string(corev1.PodRunning),
		}.String(),
	})
	if err != nil {
		return nil, nil, err
	}

	switch len(pods.Items) {
	case 0:
		// Perhaps not a valid source IP.
		// TODO: Security incident?
		return nil, nil, &NotFoundError{Reason: fmt.Sprintf("kubernetes: no pod found with IP %s", ki.ip)}
	case 1:
		// Only acceptable case.
	default:
		// Multiple pods with the same IP? This is just nonsense and we'll throw
		// out the request.
		return nil, nil, &NotFoundError{Reason: fmt.Sprintf("kubernetes: multiple pods found with IP %s (bug?)", ki.ip)}
	}

	pod := pods.Items[0]

	ns, err := ki.client.CoreV1().Namespaces().Get(pod.GetNamespace(), metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil, nil, &NotFoundError{Reason: "kubernetes: namespace of requesting pod no longer exists"}
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

		// XXX: This assumes that Tasks and Conditions have the same name. This
		// is true for us, but not for Tekton generally.
		name := pod.GetLabels()["tekton.dev/pipelineTask"]

		tr, err := ki.client.TektonV1alpha1().Conditions(ns.GetName()).Get(name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil, nil, &NotFoundError{Reason: "kubernetes: Tekton condition of requesting pod does not exist"}
		} else if err != nil {
			return nil, nil, err
		}

		// Check and copy out the annotations.
		subject = tr.GetAnnotations()[KubernetesSubjectAnnotation]

		tok, found = tr.GetAnnotations()[KubernetesTokenAnnotation]
		if !found || tok == "" {
			return nil, nil, &NotFoundError{Reason: "kubernetes: subject and token annotation not present on pod or Tekton condition"}
		}
	}

	md := &KubernetesIntermediaryMetadata{
		NamespaceUID: ns.GetUID(),
	}

	// Namespace validation.
	state.AddValidator(ValidatorFunc(func(ctx context.Context, claims *Claims) (bool, error) {
		if claims.KubernetesNamespaceUID == "" {
			log(ctx).Warn("kubernetes: no namespace UID in claims")
			return false, nil
		}

		r := md.NamespaceUID == types.UID(claims.KubernetesNamespaceUID)
		if !r {
			log(ctx).Warn("kubernetes: namespace UID of claim does not match namespace UID of pod", "claim-namespace-uid", claims.KubernetesNamespaceUID, "pod-namespace-uid", md.NamespaceUID)
		}

		return r, nil
	}))

	state.AddValidator(ValidatorFunc(func(ctx context.Context, claims *Claims) (bool, error) {
		if claims.Subject == "" {
			log(ctx).Warn("kubernetes: no subject in claims")
			return false, nil
		}

		r := subject == claims.Subject
		if !r {
			log(ctx).Warn("kubernetes: subject of claim does not match subject annotation of pod", "claim-subject", claims.Subject, "pod-subject-annotation", subject)
		}

		return r, nil
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
