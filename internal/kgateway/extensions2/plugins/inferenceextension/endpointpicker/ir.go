package endpointpicker

import (
	"maps"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	infextv1a2 "sigs.k8s.io/gateway-api-inference-extension/api/v1alpha2"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/ir"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/wellknown"
)

// inferencePool defines the internal representation of an inferencePool resource.
type inferencePool struct {
	objMeta metav1.ObjectMeta
	// podSelector is a label selector to select Pods that are members of the InferencePool.
	podSelector map[string]string
	// targetPort is the port number that should be targeted for Pods selected by Selector.
	targetPort int32
	// configRef is a reference to the extension configuration. A configRef is typically implemented
	// as a Kubernetes Service resource.
	configRef *ir.Service
}

// newInferencePool returns the internal representation of the given pool.
func newInferencePool(pool *infextv1a2.InferencePool) *inferencePool {
	if pool == nil || pool.Spec.ExtensionRef == nil {
		return nil
	}

	port := ir.ServicePort{Name: "grpc", PortNum: (int32(ir.GRPCport))}
	if pool.Spec.ExtensionRef.PortNumber != nil {
		port.PortNum = int32(*pool.Spec.ExtensionRef.PortNumber)
	}

	svcIR := &ir.Service{
		ObjectSource: ir.ObjectSource{
			Group:     infextv1a2.GroupVersion.Group,
			Kind:      wellknown.InferencePoolKind,
			Namespace: pool.Namespace,
			Name:      string(pool.Spec.ExtensionRef.Name),
		},
		Obj:   pool,
		Ports: []ir.ServicePort{port},
	}

	return &inferencePool{
		objMeta:     pool.ObjectMeta,
		podSelector: convertSelector(pool.Spec.Selector),
		targetPort:  int32(pool.Spec.TargetPortNumber),
		configRef:   svcIR,
	}
}

// In case multiple pools attached to the same resource, we sort by creation time.
func (ir *inferencePool) CreationTime() time.Time {
	return ir.objMeta.CreationTimestamp.Time
}

func (ir *inferencePool) Selector() map[string]string {
	if ir.podSelector == nil {
		return nil
	}
	return ir.podSelector
}

func (ir *inferencePool) Equals(other any) bool {
	otherPool, ok := other.(*inferencePool)
	if !ok {
		return false
	}
	return maps.EqualFunc(ir.Selector(), otherPool.Selector(), func(a, b string) bool {
		return a == b
	})
}

func convertSelector(selector map[infextv1a2.LabelKey]infextv1a2.LabelValue) map[string]string {
	result := make(map[string]string, len(selector))
	for k, v := range selector {
		result[string(k)] = string(v)
	}
	return result
}
