package obj

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DefaultNetworkPolicyDeniedIPBlocks = []string{
		"0.0.0.0/8",       // "This host on this network"
		"10.0.0.0/8",      // Private-Use
		"100.64.0.0/10",   // Shared Address Space
		"169.254.0.0/16",  // Link Local
		"172.16.0.0/12",   // Private-Use
		"192.0.0.0/24",    // IETF Protocol Assignments
		"192.0.2.0/24",    // Documentation (TEST-NET-1)
		"192.31.196.0/24", // AS112-v4
		"192.52.193.0/24", // AMT
		"192.168.0.0/16",  // Private-Use
		"192.175.48.0/24", // Direct Delegation AS112 Service
		"198.18.0.0/15",   // Benchmarking
		"198.51.100.0/24", // Documentation (TEST-NET-2)
		"203.0.113.0/24",  // Documentation (TEST-NET-3)
		"240.0.0.0/4",     // Reserved (multicast)
	}
)

type NetworkPolicy struct {
	Key    client.ObjectKey
	Object *networkingv1.NetworkPolicy
}

var _ Persister = &NetworkPolicy{}
var _ Loader = &NetworkPolicy{}
var _ Ownable = &NetworkPolicy{}

func (np *NetworkPolicy) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, np.Key, np.Object)
}

func (np *NetworkPolicy) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, np.Key, np.Object)
}

func (np *NetworkPolicy) Owned(ctx context.Context, ref *metav1.OwnerReference) {
	Own(&np.Object.ObjectMeta, ref)
}

func NewNetworkPolicy(key client.ObjectKey) *NetworkPolicy {
	return &NetworkPolicy{
		Key:    key,
		Object: &networkingv1.NetworkPolicy{},
	}
}

type networkPolicyOptions struct {
	deniedIPBlocks          []string
	systemNamespaceSelector metav1.LabelSelector
	metadataAPIPodSelector  metav1.LabelSelector
	metadataAPIPort         int
}

type NetworkPolicyOption func(opts *networkPolicyOptions)

func NetworkPolicyWithDeniedIPBlocks(blocks []string) NetworkPolicyOption {
	return func(opts *networkPolicyOptions) {
		opts.deniedIPBlocks = append([]string{}, blocks...)
	}
}

func NetworkPolicyWithSystemNamespaceSelector(selector metav1.LabelSelector) NetworkPolicyOption {
	return func(opts *networkPolicyOptions) {
		opts.systemNamespaceSelector = selector
	}
}

func NetworkPolicyWithMetadataAPIPodSelector(selector metav1.LabelSelector) NetworkPolicyOption {
	return func(opts *networkPolicyOptions) {
		opts.metadataAPIPodSelector = selector
	}
}

func NetworkPolicyWithMetadataAPIPort(port int) NetworkPolicyOption {
	return func(opts *networkPolicyOptions) {
		opts.metadataAPIPort = port
	}
}

func ConfigureNetworkPolicyForTenant(np *NetworkPolicy) {
	// The default tenant policy blocks all traffic. Additional policies are
	// additive.
	np.Object.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
			networkingv1.PolicyTypeEgress,
		},
	}
}

func ConfigureNetworkPolicyForWorkflowRun(np *NetworkPolicy, wr *WorkflowRun, opts ...NetworkPolicyOption) {
	np.Object.Spec = baseTenantWorkloadNetworkPolicySpec(wr.PodSelector(), opts)
}

func ConfigureNetworkPolicyForWebhookTrigger(np *NetworkPolicy, wt *WebhookTrigger, opts ...NetworkPolicyOption) {
	np.Object.Spec = baseTenantWorkloadNetworkPolicySpec(wt.PodSelector(), opts)

	// Allow ingress from the defined upstream.
	np.Object.Spec.Ingress = append(np.Object.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{
			{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"nebula.puppet.com/network-policy.webhook-gateway": "true",
					},
				},
				PodSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"nebula.puppet.com/network-policy.webhook-gateway": "true",
					},
				},
			},
		},
	})
}

func baseTenantWorkloadNetworkPolicySpec(podSelector metav1.LabelSelector, opts []NetworkPolicyOption) networkingv1.NetworkPolicySpec {
	npo := &networkPolicyOptions{
		deniedIPBlocks: DefaultNetworkPolicyDeniedIPBlocks,
		systemNamespaceSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				"nebula.puppet.com/network-policy.tasks": "true",
			},
		},
		metadataAPIPodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app.kubernetes.io/name":      "nebula-system",
				"app.kubernetes.io/component": "metadata-api",
			},
		},
		metadataAPIPort: 7000,
	}

	for _, opt := range opts {
		opt(npo)
	}

	return networkingv1.NetworkPolicySpec{
		PodSelector: podSelector,
		PolicyTypes: []networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
			networkingv1.PolicyTypeEgress,
		},
		Ingress: []networkingv1.NetworkPolicyIngressRule{},
		Egress: []networkingv1.NetworkPolicyEgressRule{
			{
				// Allow all external traffic except RFC 1918 space and IANA
				// special-purpose address registry.
				To: []networkingv1.NetworkPolicyPeer{
					{
						IPBlock: &networkingv1.IPBlock{
							CIDR:   "0.0.0.0/0",
							Except: npo.deniedIPBlocks,
						},
					},
				},
			},
			{
				// Allow access to the metadata API.
				To: []networkingv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &npo.systemNamespaceSelector,
						PodSelector:       &npo.metadataAPIPodSelector,
					},
				},
				Ports: []networkingv1.NetworkPolicyPort{
					{
						Protocol: func(p corev1.Protocol) *corev1.Protocol { return &p }(corev1.ProtocolTCP),
						Port:     func(i intstr.IntOrString) *intstr.IntOrString { return &i }(intstr.FromInt(npo.metadataAPIPort)),
					},
				},
			},
		},
	}
}
