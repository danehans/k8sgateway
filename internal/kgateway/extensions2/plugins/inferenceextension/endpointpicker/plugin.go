package endpointpicker

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"istio.io/istio/pkg/kube/kclient"
	"istio.io/istio/pkg/kube/krt"
	infextv1a1 "sigs.k8s.io/gateway-api-inference-extension/api/v1alpha1"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/extensions2/common"
	extensionsplug "github.com/kgateway-dev/kgateway/v2/internal/kgateway/extensions2/plugin"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/ir"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/utils/krtutil"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/wellknown"
)

func NewPlugin(ctx context.Context, commoncol *common.CommonCollections) extensionsplug.Plugin {
	poolClient := kclient.New[*infextv1a1.InferencePool](commoncol.Client)
	pools := krt.WrapClient(poolClient, commoncol.KrtOpts.ToOptions("InferencePools")...)
	return NewPluginFromCollections(ctx, commoncol.KrtOpts, pools)
}

func NewPluginFromCollections(
	ctx context.Context,
	krtOpts krtutil.KrtOptions,
	pools krt.Collection[*infextv1a1.InferencePool],
) extensionsplug.Plugin {
	gk := schema.GroupKind{
		Group: infextv1a1.GroupVersion.Group,
		Kind:  wellknown.InferencePoolKind,
	}

	infPoolUpstreams := krt.NewManyCollection(pools, func(kctx krt.HandlerContext, pool *infextv1a1.InferencePool) []ir.Upstream {
		uss := []ir.Upstream{}
		uss = append(uss, ir.Upstream{
			ObjectSource: ir.ObjectSource{
				Kind:      gk.Kind,
				Group:     gk.Group,
				Namespace: pool.Namespace,
				Name:      pool.Name,
			},
			Obj:               pool,
			Port:              pool.Spec.TargetPortNumber,
			GvPrefix:          "endpoint-picker",
			// Not need to integrate with destination rule at this time.
			CanonicalHostname: "",
		})
		return uss
	}, krtOpts.ToOptions("EndpointPickerUpstreams")...)

	return extensionsplug.Plugin{
		ContributesUpstreams: map[schema.GroupKind]extensionsplug.UpstreamPlugin{
			gk: {
				UpstreamInit: ir.UpstreamInit{
					InitUpstream: processUpstream,
				},
				// The endpoint picker extension provides the endpoints.
				Endpoints: nil,
				Upstreams: infPoolUpstreams,
			},
		},
	}
}

func processUpstream(ctx context.Context, in ir.Upstream, out *envoy_config_cluster_v3.Cluster) {
	// Set cluster type to ORIGINAL_DST
	out.ClusterDiscoveryType = &envoy_config_cluster_v3.Cluster_Type{
		Type: envoy_config_cluster_v3.Cluster_ORIGINAL_DST,
	}

	// Set connect timeout to 1000 seconds.
	// TODO [danehans]: Figure out an API that can be used to set this value.
	out.ConnectTimeout = durationpb.New(1000 * time.Second)

	// Use CLUSTER_PROVIDED load balancing.
	out.LbPolicy = envoy_config_cluster_v3.Cluster_CLUSTER_PROVIDED

	// Configure circuit breakers with a single threshold.
	// TODO [danehans]: Figure out an API that can be used to set these values.
	out.CircuitBreakers = &envoy_config_cluster_v3.CircuitBreakers{
		Thresholds: []*envoy_config_cluster_v3.CircuitBreakers_Thresholds{
			{
				MaxConnections:     wrapperspb.UInt32(40000),
				MaxPendingRequests: wrapperspb.UInt32(40000),
				MaxRequests:        wrapperspb.UInt32(40000),
			},
		},
	}

	// If OriginalDstLbConfig is not available on Cluster,
	// encode the configuration as a typed extension.
	// Note: The type URL will be "type.googleapis.com/envoy.config.cluster.v3.Cluster_OriginalDstLbConfig".
	lbConfig := &envoy_config_cluster_v3.Cluster_OriginalDstLbConfig{
		UseHttpHeader:  true,
		HttpHeaderName: "x-gateway-destination-endpoint",
	}
	anyLbConfig, err := anypb.New(lbConfig)
	if err != nil {
		// handle error appropriately
		return
	}
	out.TypedExtensionProtocolOptions = map[string]*anypb.Any{
		"envoy.lb": anyLbConfig,
	}
}
