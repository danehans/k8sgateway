// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// InferenceExtensionApplyConfiguration represents a declarative configuration of the InferenceExtension type for use
// with apply.
type InferenceExtensionApplyConfiguration struct {
	EndpointPicker *EndpointPickerExtensionApplyConfiguration `json:"endpointPicker,omitempty"`
}

// InferenceExtensionApplyConfiguration constructs a declarative configuration of the InferenceExtension type for use with
// apply.
func InferenceExtension() *InferenceExtensionApplyConfiguration {
	return &InferenceExtensionApplyConfiguration{}
}

// WithEndpointPicker sets the EndpointPicker field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the EndpointPicker field is set to the value of the last call.
func (b *InferenceExtensionApplyConfiguration) WithEndpointPicker(value *EndpointPickerExtensionApplyConfiguration) *InferenceExtensionApplyConfiguration {
	b.EndpointPicker = value
	return b
}
