package ir

import (
	"encoding/json"

	"istio.io/istio/pkg/kube/krt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// GRPCPort is the default port number for the gRPC service.
	GRPCPort = 9002
)

// Service defines an internal representation of a service.
type Service struct {
	// ObjectSource is a reference to the source object. Sometimes the group and kind are not
	// populated from api-server, so set them explicitly here, and pass this around as the reference.
	ObjectSource `json:",inline"`

	// Obj is the original object. Opaque to us other than metadata.
	Obj metav1.Object

	// Ports is a list of ports exposed by the service.
	Ports []ServicePort
}

// ServicePort is an exposed post of a service.
type ServicePort struct {
	// Name is the name of the port.
	Name string
	// PortNum is the port number used to expose the service port.
	PortNum int32
}

func (r Service) ResourceName() string {
	return r.ObjectSource.ResourceName()
}

func (r Service) Equals(in Service) bool {
	return r.ObjectSource.Equals(in.ObjectSource) && versionEquals(r.Obj, in.Obj)
}

var _ krt.ResourceNamer = Service{}
var _ krt.Equaler[Service] = Service{}
var _ json.Marshaler = Service{}

func (l Service) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Group     string
		Kind      string
		Name      string
		Namespace string
		Ports     []ServicePort
	}{
		Group:     l.Group,
		Kind:      l.Kind,
		Namespace: l.Namespace,
		Name:      l.Name,
		Ports:     l.Ports,
	})
}
