package ir

import (
	"maps"
)

type InferencePool struct {
	// PodSelector is a label selector to select Pods that are members of the InferencePool.
	PodSelector map[string]string
	// TargetPort is the port number that should be targeted for Pods selected by Selector.
	TargetPort int32
	// ConfigRef is a reference to the extension configuration. A ConfigRef is typically implemented
	// as a Kubernetes Service resource.
	ConfigRef *Service
}

func (ir *InferencePool) Selector() map[string]string {
	if ir.PodSelector == nil {
		return nil
	}
	return ir.PodSelector
}

func (ir *InferencePool) Equals(other any) bool {
	otherPool, ok := other.(*InferencePool)
	if !ok {
		return false
	}
	return maps.EqualFunc(ir.Selector(), otherPool.Selector(), func(a, b string) bool {
		return a == b
	})
}
