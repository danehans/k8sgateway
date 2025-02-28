package endpointpicker

import (
	"context"
	"testing"

	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyendpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/fgrosse/zaptest"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/contextutils"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/kube/krt/krttest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	infextv1a1 "sigs.k8s.io/gateway-api-inference-extension/api/v1alpha1"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/ir"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/krtcollections"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/utils/krtutil"
)

func TestEndpointsFromInferencePool(t *testing.T) {
	logger := zaptest.Logger(t)
	contextutils.SetFallbackLogger(logger.Sugar())

	testCases := []struct {
		name    string
		inputs  []any
		backend ir.BackendObjectIR
		result  func(ir.BackendObjectIR) *ir.EndpointsForBackend
	}{
		{
			name: "one matching pod",
			inputs: []any{
				&infextv1a1.InferencePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "inf-pool",
						Namespace: "ns",
					},
					Spec: infextv1a1.InferencePoolSpec{
						TargetPortNumber: 8080,
						Selector: map[infextv1a1.LabelKey]infextv1a1.LabelValue{
							"app": "inference",
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns",
						Labels: map[string]string{
							"app": "inference", // Matches inferencepool selector
						},
					},
					Spec: corev1.PodSpec{
						NodeName: "node1",
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						PodIP: "1.2.3.4",
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Labels: map[string]string{
							corev1.LabelTopologyRegion: "region1",
							corev1.LabelTopologyZone:   "zone1",
						},
					},
				},
			},
			backend: ir.BackendObjectIR{
				ObjectSource: ir.ObjectSource{
					Namespace: "ns",
					Name:      "inf-pool",
					Group:     infextv1a1.GroupVersion.Group,
					Kind:      "InferencePool",
				},
				Port: 8080,
				Obj: &infextv1a1.InferencePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "inf-pool",
						Namespace: "ns",
					},
					Spec: infextv1a1.InferencePoolSpec{
						TargetPortNumber: 8080,
						Selector: map[infextv1a1.LabelKey]infextv1a1.LabelValue{
							"app": "inference",
						},
					},
				},
			},
			result: func(us ir.BackendObjectIR) *ir.EndpointsForBackend {
				emd := ir.EndpointWithMd{
					LbEndpoint: &envoyendpointv3.LbEndpoint{
						LoadBalancingWeight: wrapperspb.UInt32(1),
						HostIdentifier: &envoyendpointv3.LbEndpoint_Endpoint{
							Endpoint: &envoyendpointv3.Endpoint{
								Address: &envoycorev3.Address{
									Address: &envoycorev3.Address_SocketAddress{
										SocketAddress: &envoycorev3.SocketAddress{
											Address: "1.2.3.4",
											PortSpecifier: &envoycorev3.SocketAddress_PortValue{
												PortValue: 8080,
											},
										},
									},
								},
							},
						},
					},
					EndpointMd: ir.EndpointMetadata{
						Labels: map[string]string{
							"app":                      "inference",
							corev1.LabelTopologyRegion: "region1",
							corev1.LabelTopologyZone:   "zone1",
						},
					},
				}
				result := ir.NewEndpointsForBackend(us)
				result.Add(ir.PodLocality{
					Region: "region1",
					Zone:   "zone1",
				}, emd)
				return result
			},
		},
		{
			name: "no matching pods",
			inputs: []any{
				&infextv1a1.InferencePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "inf-pool",
						Namespace: "ns",
					},
					Spec: infextv1a1.InferencePoolSpec{
						TargetPortNumber: 8080,
						Selector: map[infextv1a1.LabelKey]infextv1a1.LabelValue{
							"app": "non-existent",
						},
					},
				},
			},
			backend: ir.BackendObjectIR{
				ObjectSource: ir.ObjectSource{
					Namespace: "ns",
					Name:      "inf-pool",
					Group:     infextv1a1.GroupVersion.Group,
					Kind:      "InferencePool",
				},
				Port: 8080,
				Obj: &infextv1a1.InferencePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "inf-pool",
						Namespace: "ns",
					},
					Spec: infextv1a1.InferencePoolSpec{
						TargetPortNumber: 8080,
						Selector: map[infextv1a1.LabelKey]infextv1a1.LabelValue{
							"app": "non-existent",
						},
					},
				},
			},
			result: func(us ir.BackendObjectIR) *ir.EndpointsForBackend {
				// No matching pods, so result should be empty.
				return ir.NewEndpointsForBackend(us)
			},
		},
		{
			name: "multiple matching pods",
			inputs: []any{
				&infextv1a1.InferencePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "inf-pool",
						Namespace: "ns",
					},
					Spec: infextv1a1.InferencePoolSpec{
						TargetPortNumber: 8080,
						Selector: map[infextv1a1.LabelKey]infextv1a1.LabelValue{
							"app": "inference",
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns",
						Labels: map[string]string{
							"app": "inference",
						},
					},
					Spec: corev1.PodSpec{
						NodeName: "node1",
					},
					Status: corev1.PodStatus{
						PodIP: "1.2.3.4",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns",
						Labels: map[string]string{
							"app": "inference",
						},
					},
					Spec: corev1.PodSpec{
						NodeName: "node2",
					},
					Status: corev1.PodStatus{
						PodIP: "1.2.3.5",
					},
				},
			},
			backend: ir.BackendObjectIR{
				ObjectSource: ir.ObjectSource{
					Namespace: "ns",
					Name:      "inf-pool",
					Group:     infextv1a1.GroupVersion.Group,
					Kind:      "InferencePool",
				},
				Port: 8080,
				Obj: &infextv1a1.InferencePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "inf-pool",
						Namespace: "ns",
					},
					Spec: infextv1a1.InferencePoolSpec{
						TargetPortNumber: 8080,
						Selector: map[infextv1a1.LabelKey]infextv1a1.LabelValue{
							"app": "inference",
						},
					},
				},
			},
			result: func(us ir.BackendObjectIR) *ir.EndpointsForBackend {
				// Expect both pods to be included.
				result := ir.NewEndpointsForBackend(us)
				result.Add(ir.PodLocality{}, ir.EndpointWithMd{
					LbEndpoint: krtcollections.CreateLBEndpoint("1.2.3.4", 8080, map[string]string{"app": "inference"}, true),
					EndpointMd: ir.EndpointMetadata{
						Labels: map[string]string{"app": "inference"},
					},
				})
				result.Add(ir.PodLocality{}, ir.EndpointWithMd{
					LbEndpoint: krtcollections.CreateLBEndpoint("1.2.3.5", 8080, map[string]string{"app": "inference"}, true),
					EndpointMd: ir.EndpointMetadata{
						Labels: map[string]string{"app": "inference"},
					},
				})
				return result
			},
		},
		{
			name: "pods in different namespaces",
			inputs: []any{
				&infextv1a1.InferencePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "inf-pool",
						Namespace: "ns1",
					},
					Spec: infextv1a1.InferencePoolSpec{
						TargetPortNumber: 8080,
						Selector: map[infextv1a1.LabelKey]infextv1a1.LabelValue{
							"app": "inference",
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns2",
						Labels: map[string]string{
							"app": "inference",
						},
					},
					Status: corev1.PodStatus{
						PodIP: "1.2.3.4",
					},
				},
			},
			backend: ir.BackendObjectIR{
				ObjectSource: ir.ObjectSource{
					Namespace: "ns1",
					Name:      "inf-pool",
					Group:     infextv1a1.GroupVersion.Group,
					Kind:      "InferencePool",
				},
				Port: 8080,
				Obj: &infextv1a1.InferencePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "inf-pool",
						Namespace: "ns1",
					},
					Spec: infextv1a1.InferencePoolSpec{
						TargetPortNumber: 8080,
						Selector: map[infextv1a1.LabelKey]infextv1a1.LabelValue{
							"app": "inference",
						},
					},
				},
			},
			result: func(us ir.BackendObjectIR) *ir.EndpointsForBackend {
				// No pods should be selected since they are in a different namespace.
				return ir.NewEndpointsForBackend(us)
			},
		},
		{
			name: "pods with no IPs",
			inputs: []any{
				&infextv1a1.InferencePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "inf-pool",
						Namespace: "ns",
					},
					Spec: infextv1a1.InferencePoolSpec{
						TargetPortNumber: 8080,
						Selector: map[infextv1a1.LabelKey]infextv1a1.LabelValue{
							"app": "inference",
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns",
						Labels: map[string]string{
							"app": "inference",
						},
					},
					Status: corev1.PodStatus{
						PodIP: "",
					},
				},
			},
			backend: ir.BackendObjectIR{
				ObjectSource: ir.ObjectSource{
					Namespace: "ns",
					Name:      "inf-pool",
					Group:     infextv1a1.GroupVersion.Group,
					Kind:      "InferencePool",
				},
				Port: 8080,
				Obj: &infextv1a1.InferencePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "inf-pool",
						Namespace: "ns",
					},
					Spec: infextv1a1.InferencePoolSpec{
						TargetPortNumber: 8080,
						Selector: map[infextv1a1.LabelKey]infextv1a1.LabelValue{
							"app": "inference",
						},
					},
				},
			},
			result: func(us ir.BackendObjectIR) *ir.EndpointsForBackend {
				// No endpoints should be created since the pod has no IP.
				return ir.NewEndpointsForBackend(us)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			mock := krttest.NewMock(t, tc.inputs)

			// Initialize collections
			nodes := krtcollections.NewNodeMetadataCollection(krttest.GetMockCollection[*corev1.Node](mock))
			pods := krtcollections.NewLocalityPodsCollection(nodes, krttest.GetMockCollection[*corev1.Pod](mock), krtutil.KrtOptions{})
			pods.WaitUntilSynced(context.Background().Done())

			// Create InferencePool collection
			infPools := krttest.GetMockCollection[*infextv1a1.InferencePool](mock)
			infPoolBackendObjectIR := krt.NewCollection(infPools, func(kctx krt.HandlerContext, pool *infextv1a1.InferencePool) *ir.BackendObjectIR {
				return &ir.BackendObjectIR{
					ObjectSource: ir.ObjectSource{
						Namespace: pool.Namespace,
						Name:      pool.Name,
						Group:     infextv1a1.GroupVersion.Group,
						Kind:      "InferencePool",
					},
					Obj:  pool,
					Port: pool.Spec.TargetPortNumber,
				}
			}, krtutil.KrtOptions{}.ToOptions("InfPoolBackendObjectIRs")...)

			// Initialize InferencePool-based endpoint collection
			infPoolInputs := newInfPoolEndpointsInputs(krtutil.KrtOptions{}, infPoolBackendObjectIR, pods)

			ctx := context.Background()
			builder := transformInfPoolEndpoints(ctx, infPoolInputs)

			// Run the test transformation
			eps := builder(krt.TestingDummyContext{}, tc.backend)
			res := tc.result(tc.backend)

			if eps == nil && res == nil {
				return // Both are nil, test passes
			}

			g.Expect(eps).ToNot(BeNil(), "Expected non-nil endpoints, but got nil")
			g.Expect(res).ToNot(BeNil(), "Expected nil endpoints, but got non-nil")
			g.Expect(eps.Equals(*res)).To(BeTrue(), "expected %v, got %v", res, eps)
		})
	}
}
