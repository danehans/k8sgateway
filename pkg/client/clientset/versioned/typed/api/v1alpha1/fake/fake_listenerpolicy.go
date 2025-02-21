// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	gentype "k8s.io/client-go/gentype"

	apiv1alpha1 "github.com/kgateway-dev/kgateway/v2/api/applyconfiguration/api/v1alpha1"
	v1alpha1 "github.com/kgateway-dev/kgateway/v2/api/v1alpha1"
	typedapiv1alpha1 "github.com/kgateway-dev/kgateway/v2/pkg/client/clientset/versioned/typed/api/v1alpha1"
)

// fakeListenerPolicies implements ListenerPolicyInterface
type fakeListenerPolicies struct {
	*gentype.FakeClientWithListAndApply[*v1alpha1.ListenerPolicy, *v1alpha1.ListenerPolicyList, *apiv1alpha1.ListenerPolicyApplyConfiguration]
	Fake *FakeGatewayV1alpha1
}

func newFakeListenerPolicies(fake *FakeGatewayV1alpha1, namespace string) typedapiv1alpha1.ListenerPolicyInterface {
	return &fakeListenerPolicies{
		gentype.NewFakeClientWithListAndApply[*v1alpha1.ListenerPolicy, *v1alpha1.ListenerPolicyList, *apiv1alpha1.ListenerPolicyApplyConfiguration](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("listenerpolicies"),
			v1alpha1.SchemeGroupVersion.WithKind("ListenerPolicy"),
			func() *v1alpha1.ListenerPolicy { return &v1alpha1.ListenerPolicy{} },
			func() *v1alpha1.ListenerPolicyList { return &v1alpha1.ListenerPolicyList{} },
			func(dst, src *v1alpha1.ListenerPolicyList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.ListenerPolicyList) []*v1alpha1.ListenerPolicy {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1alpha1.ListenerPolicyList, items []*v1alpha1.ListenerPolicy) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
