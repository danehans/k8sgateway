package krtcollections

import (
	"istio.io/istio/pkg/kube/krt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	infextv1a1 "sigs.k8s.io/gateway-api-inference-extension/api/v1alpha1"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/ir"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/wellknown"
)

type ServiceIndex struct {
	services map[schema.GroupKind]krt.Collection[ir.Service]
}

func NewServiceIndex(services map[schema.GroupKind]krt.Collection[ir.Service]) *ServiceIndex {
	return &ServiceIndex{services: services}
}

func (s *ServiceIndex) HasSynced() bool {
	for _, col := range s.services {
		if !col.HasSynced() {
			return false
		}
	}
	return true
}

func (s *ServiceIndex) GetSvcForInferPool(kctx krt.HandlerContext, pool *infextv1a1.InferencePool) *ir.Service {
	if pool == nil || pool.Spec.ExtensionRef == nil {
		// TODO [danehans]: Add logging
		return nil
	}

	refGroup := ""
	ref := *pool.Spec.ExtensionRef
	if ref.Group != nil && *ref.Group != "" {
		// TODO [danehans]: Add logging
		return nil
	}
	refKind := wellknown.ServiceKind
	if ref.Kind != nil && *ref.Kind != "" {
		// TODO [danehans]: Add logging
		return nil
	}

	// Get the krt Service collection
	gk := schema.GroupKind{Group: refGroup, Kind: *ref.Kind}
	col := s.services[gk]
	if col == nil {
		// TODO [danehans]: Add logging
		return nil
	}

	// Create the object source used for filtering by name when fetching Services from the collection.
	src := ir.ObjectSource{
		Group:     refGroup,
		Kind:      refKind,
		Namespace: pool.Namespace,
		Name:      pool.Name,
	}

	// Fetch the Service from the krt collection, filtering based on object source name.
	ret := krt.FetchOne(kctx, col, krt.FilterKey(src.ResourceName()))
	if ret == nil {
		// TODO [danehans]: Add logging
		return nil
	}

	return ret
}
