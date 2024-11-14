package wellknown

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiv1 "sigs.k8s.io/gateway-api/apis/v1"
	apiv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	apiv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const (
	// Group string for Gateway API resources
	GatewayGroup = apiv1.GroupName

	// Kind string for k8s service
	ServiceKind = "Service"

	// Kind string for HTTPRoute resource
	HTTPRouteKind = "HTTPRoute"

	// Kind string for TCPRoute resource
	TCPRouteKind = "TCPRoute"

	// GRPCRouteKind is the name of the TCPRoute kind
	GRPCRouteKind = "GRPCRoute"

	// Kind string for Gateway resource
	GatewayKind = "Gateway"

	// Kind string for GatewayClass resource
	GatewayClassKind = "GatewayClass"

	// Kind string for ReferenceGrant resource
	ReferenceGrantKind = "ReferenceGrant"

	// Kind strings for Gateway API list types
	HTTPRouteListKind      = "HTTPRouteList"
	GatewayListKind        = "GatewayList"
	GatewayClassListKind   = "GatewayClassList"
	ReferenceGrantListKind = "ReferenceGrantList"
)

var (
	// SupportedVersions specifies the supported versions of Gateway API.
	SupportedVersions = []string{"v1.0.0", "v1.1.0", "v1.2.0"}

	GatewayGVK = schema.GroupVersionKind{
		Group:   GatewayGroup,
		Version: apiv1.GroupVersion.Version,
		Kind:    GatewayKind,
	}
	GatewayClassGVK = schema.GroupVersionKind{
		Group:   GatewayGroup,
		Version: apiv1.GroupVersion.Version,
		Kind:    GatewayClassKind,
	}
	HTTPRouteGVK = schema.GroupVersionKind{
		Group:   GatewayGroup,
		Version: apiv1.GroupVersion.Version,
		Kind:    HTTPRouteKind,
	}
	ReferenceGrantGVK = schema.GroupVersionKind{
		Group:   GatewayGroup,
		Version: apiv1beta1.GroupVersion.Version,
		Kind:    ReferenceGrantKind,
	}

	GatewayListGVK = schema.GroupVersionKind{
		Group:   GatewayGroup,
		Version: apiv1.GroupVersion.Version,
		Kind:    GatewayListKind,
	}
	GatewayClassListGVK = schema.GroupVersionKind{
		Group:   GatewayGroup,
		Version: apiv1.GroupVersion.Version,
		Kind:    GatewayClassListKind,
	}
	HTTPRouteListGVK = schema.GroupVersionKind{
		Group:   GatewayGroup,
		Version: apiv1.GroupVersion.Version,
		Kind:    HTTPRouteListKind,
	}
	HTCPRouteListGVK = schema.GroupVersionKind{
		Group:   GatewayGroup,
		Version: apiv1alpha2.GroupVersion.Version,
		Kind:    HTTPRouteListKind,
	}
	ReferenceGrantListGVK = schema.GroupVersionKind{
		Group:   GatewayGroup,
		Version: apiv1beta1.GroupVersion.Version,
		Kind:    ReferenceGrantListKind,
	}
)
