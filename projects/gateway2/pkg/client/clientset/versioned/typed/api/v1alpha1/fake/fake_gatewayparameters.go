// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	apiv1alpha1 "github.com/solo-io/gloo/projects/gateway2/api/applyconfiguration/api/v1alpha1"
	v1alpha1 "github.com/solo-io/gloo/projects/gateway2/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGatewayParameterses implements GatewayParametersInterface
type FakeGatewayParameterses struct {
	Fake *FakeGatewayV1alpha1
	ns   string
}

var gatewayparametersesResource = v1alpha1.SchemeGroupVersion.WithResource("gatewayparameterses")

var gatewayparametersesKind = v1alpha1.SchemeGroupVersion.WithKind("GatewayParameters")

// Get takes name of the gatewayParameters, and returns the corresponding gatewayParameters object, and an error if there is any.
func (c *FakeGatewayParameterses) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.GatewayParameters, err error) {
	emptyResult := &v1alpha1.GatewayParameters{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(gatewayparametersesResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.GatewayParameters), err
}

// List takes label and field selectors, and returns the list of GatewayParameterses that match those selectors.
func (c *FakeGatewayParameterses) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.GatewayParametersList, err error) {
	emptyResult := &v1alpha1.GatewayParametersList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(gatewayparametersesResource, gatewayparametersesKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.GatewayParametersList{ListMeta: obj.(*v1alpha1.GatewayParametersList).ListMeta}
	for _, item := range obj.(*v1alpha1.GatewayParametersList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested gatewayParameterses.
func (c *FakeGatewayParameterses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(gatewayparametersesResource, c.ns, opts))

}

// Create takes the representation of a gatewayParameters and creates it.  Returns the server's representation of the gatewayParameters, and an error, if there is any.
func (c *FakeGatewayParameterses) Create(ctx context.Context, gatewayParameters *v1alpha1.GatewayParameters, opts v1.CreateOptions) (result *v1alpha1.GatewayParameters, err error) {
	emptyResult := &v1alpha1.GatewayParameters{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(gatewayparametersesResource, c.ns, gatewayParameters, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.GatewayParameters), err
}

// Update takes the representation of a gatewayParameters and updates it. Returns the server's representation of the gatewayParameters, and an error, if there is any.
func (c *FakeGatewayParameterses) Update(ctx context.Context, gatewayParameters *v1alpha1.GatewayParameters, opts v1.UpdateOptions) (result *v1alpha1.GatewayParameters, err error) {
	emptyResult := &v1alpha1.GatewayParameters{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(gatewayparametersesResource, c.ns, gatewayParameters, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.GatewayParameters), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeGatewayParameterses) UpdateStatus(ctx context.Context, gatewayParameters *v1alpha1.GatewayParameters, opts v1.UpdateOptions) (result *v1alpha1.GatewayParameters, err error) {
	emptyResult := &v1alpha1.GatewayParameters{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceActionWithOptions(gatewayparametersesResource, "status", c.ns, gatewayParameters, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.GatewayParameters), err
}

// Delete takes name of the gatewayParameters and deletes it. Returns an error if one occurs.
func (c *FakeGatewayParameterses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(gatewayparametersesResource, c.ns, name, opts), &v1alpha1.GatewayParameters{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGatewayParameterses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionActionWithOptions(gatewayparametersesResource, c.ns, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.GatewayParametersList{})
	return err
}

// Patch applies the patch and returns the patched gatewayParameters.
func (c *FakeGatewayParameterses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.GatewayParameters, err error) {
	emptyResult := &v1alpha1.GatewayParameters{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(gatewayparametersesResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.GatewayParameters), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied gatewayParameters.
func (c *FakeGatewayParameterses) Apply(ctx context.Context, gatewayParameters *apiv1alpha1.GatewayParametersApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.GatewayParameters, err error) {
	if gatewayParameters == nil {
		return nil, fmt.Errorf("gatewayParameters provided to Apply must not be nil")
	}
	data, err := json.Marshal(gatewayParameters)
	if err != nil {
		return nil, err
	}
	name := gatewayParameters.Name
	if name == nil {
		return nil, fmt.Errorf("gatewayParameters.Name must be provided to Apply")
	}
	emptyResult := &v1alpha1.GatewayParameters{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(gatewayparametersesResource, c.ns, *name, types.ApplyPatchType, data, opts.ToPatchOptions()), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.GatewayParameters), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeGatewayParameterses) ApplyStatus(ctx context.Context, gatewayParameters *apiv1alpha1.GatewayParametersApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.GatewayParameters, err error) {
	if gatewayParameters == nil {
		return nil, fmt.Errorf("gatewayParameters provided to Apply must not be nil")
	}
	data, err := json.Marshal(gatewayParameters)
	if err != nil {
		return nil, err
	}
	name := gatewayParameters.Name
	if name == nil {
		return nil, fmt.Errorf("gatewayParameters.Name must be provided to Apply")
	}
	emptyResult := &v1alpha1.GatewayParameters{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(gatewayparametersesResource, c.ns, *name, types.ApplyPatchType, data, opts.ToPatchOptions(), "status"), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.GatewayParameters), err
}