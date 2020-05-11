package authenticate

import (
	"context"
	"fmt"
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	KubernetesTokenAnnotation   = "relay.sh/token"
	KubernetesSubjectAnnotation = "relay.sh/subject"
)

type KubernetesIntermediaryMetadata struct {
	NamespaceUID types.UID
}

type KubernetesChainIntermediaryFunc func(ctx context.Context, raw Raw, md *KubernetesIntermediaryMetadata) (Intermediary, error)

// KubernetesIntermediary looks up a pod by IP and reads the value of an
// annotation as the authentication credential.
type KubernetesIntermediary struct {
	client kubernetes.Interface
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
	if err != nil {
		return nil, nil, err
	}

	tok, found := pod.GetAnnotations()[KubernetesTokenAnnotation]
	if !found || tok == "" {
		// Request from a pod that cannot be authenticated.
		return nil, nil, ErrNotFound
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

		return pod.GetAnnotations()[KubernetesSubjectAnnotation] == claims.Subject, nil
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

func NewKubernetesIntermediary(client kubernetes.Interface, ip net.IP) *KubernetesIntermediary {
	return &KubernetesIntermediary{
		client: client,
		ip:     ip,
	}
}
