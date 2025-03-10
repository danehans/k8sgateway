package endpointpicker

import (
	"context"
	"fmt"
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	extprocv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ext_proc/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	upstreamsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"istio.io/istio/pkg/kube/krt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	infextv1a2 "sigs.k8s.io/gateway-api-inference-extension/api/v1alpha2"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/extensions2/common"
	extplug "github.com/kgateway-dev/kgateway/v2/internal/kgateway/extensions2/plugin"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/ir"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/plugins"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/utils"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/utils/krtutil"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/wellknown"

	"github.com/solo-io/go-utils/contextutils"
)

func NewPlugin(ctx context.Context, commonCol *common.CommonCollections) extplug.Plugin {
	poolGVR := schema.GroupVersionResource{
		Group:    infextv1a2.GroupVersion.Group,
		Version:  infextv1a2.GroupVersion.Version,
		Resource: "inferencepools",
	}

	poolCol := krtutil.SetupCollectionDynamic[infextv1a2.InferencePool](
		ctx,
		commonCol.Client,
		poolGVR,
		commonCol.KrtOpts.ToOptions("InferencePools")...,
	)

	return NewPluginFromCollections(ctx, commonCol, poolCol)
}

func NewPluginFromCollections(
	ctx context.Context,
	commonCol *common.CommonCollections,
	poolCol krt.Collection[*infextv1a2.InferencePool],
) extplug.Plugin {
	// The InferencePool group kind used by the BackendObjectIR and the ContributesBackendObjectIRs plugin.
	gk := schema.GroupKind{
		Group: infextv1a2.GroupVersion.Group,
		Kind:  wellknown.InferencePoolKind,
	}

	backendCol := krt.NewCollection(poolCol, func(kctx krt.HandlerContext, pool *infextv1a2.InferencePool) *ir.BackendObjectIR {
		// Create a BackendObjectIR IR representation from the given InferencePool.
		return &ir.BackendObjectIR{
			ObjectSource: ir.ObjectSource{
				Kind:      gk.Kind,
				Group:     gk.Group,
				Namespace: pool.Namespace,
				Name:      pool.Name,
			},
			Obj:               pool,
			Port:              pool.Spec.TargetPortNumber,
			GvPrefix:          "endpoint-picker",
			CanonicalHostname: "",
			ObjIr:             ir.NewInferencePool(pool),
		}
	}, commonCol.KrtOpts.ToOptions("InferencePoolIR")...)

	policyCol := krt.NewCollection(poolCol, func(krtctx krt.HandlerContext, pool *infextv1a2.InferencePool) *ir.PolicyWrapper {
		// Create a PolicyWrapper IR representation from the given InferencePool.
		return &ir.PolicyWrapper{
			ObjectSource: ir.ObjectSource{
				Group:     gk.Group,
				Kind:      gk.Kind,
				Namespace: pool.Namespace,
				Name:      pool.Name,
			},
			Policy:   pool,
			PolicyIR: ir.NewInferencePool(pool),
		}
	})

	// Return a plugin that contributes a policy and backend.
	return extplug.Plugin{
		ContributesBackends: map[schema.GroupKind]extplug.BackendPlugin{
			gk: {
				Backends: backendCol,
				BackendInit: ir.BackendInit{
					InitBackend: processBackendObjectIR,
				},
			},
		},
		ContributesPolicies: map[schema.GroupKind]extplug.PolicyPlugin{
			gk: {
				Name:                      "endpointpicker",
				Policies:                  policyCol,
				NewGatewayTranslationPass: newPolicyPlugin,
			},
		},
	}
}

// processBackendObjectIR processes the given BackendObjectIR into an Envoy cluster.
func processBackendObjectIR(ctx context.Context, in ir.BackendObjectIR, out *clusterv3.Cluster) {
	// Large timeout based on upstream working config.
	out.ConnectTimeout = durationpb.New(1000 * time.Second)

	// Set the cluster type to ORIGINAL_DST.
	out.ClusterDiscoveryType = &clusterv3.Cluster_Type{
		Type: clusterv3.Cluster_ORIGINAL_DST,
	}
	out.LbPolicy = clusterv3.Cluster_CLUSTER_PROVIDED

	// Use the headers added by the endpoint picker extension.
	out.LbConfig = &clusterv3.Cluster_OriginalDstLbConfig_{
		OriginalDstLbConfig: &clusterv3.Cluster_OriginalDstLbConfig{
			UseHttpHeader:  true,
			HttpHeaderName: "x-gateway-destination-endpoint",
		},
	}

	// Circuit breakers based on upstream working config.
	out.CircuitBreakers = &clusterv3.CircuitBreakers{
		Thresholds: []*clusterv3.CircuitBreakers_Thresholds{
			{
				MaxConnections:     wrapperspb.UInt32(40000),
				MaxPendingRequests: wrapperspb.UInt32(40000),
				MaxRequests:        wrapperspb.UInt32(40000),
			},
		},
	}

	out.Name = clusterNameOriginalDst(in.Name, in.Namespace)
}

// policyTranslation implements ir.ProxyTranslationPass. It collects any references to InferencePools,
// then in ResourcesToAdd() returns both the “ext_proc” cluster (STRICT_DNS) and “original_dst” cluster (ORIGINAL_DST).
type policyTranslation struct {
	usedPool *ir.InferencePool
}

func newPolicyPlugin(ctx context.Context, tctx ir.GwTranslationCtx) ir.ProxyTranslationPass {
	return &policyTranslation{}
}

func (p *policyTranslation) Name() string {
	return "endpoint-picker"
}

// No-op for these standard pass methods
func (p *policyTranslation) ApplyListenerPlugin(ctx context.Context, lctx *ir.ListenerContext, out *listenerv3.Listener) {
}
func (p *policyTranslation) ApplyHCM(ctx context.Context, hctx *ir.HcmContext, out *hcmv3.HttpConnectionManager) error {
	return nil
}
func (p *policyTranslation) NetworkFilters(ctx context.Context) ([]plugins.StagedNetworkFilter, error) {
	return nil, nil
}
func (p *policyTranslation) UpstreamHttpFilters(ctx context.Context) ([]plugins.StagedUpstreamHttpFilter, error) {
	return nil, nil
}
func (p *policyTranslation) ApplyVhostPlugin(ctx context.Context, vctx *ir.VirtualHostContext, out *routev3.VirtualHost) {
}
func (p *policyTranslation) ApplyForRoute(ctx context.Context, rctx *ir.RouteContext, out *routev3.Route) error {
	return nil
}
func (p *policyTranslation) ApplyRouteConfigPlugin(
	ctx context.Context,
	pCtx *ir.RouteConfigContext,
	out *routev3.RouteConfiguration,
) {
}
func (p *policyTranslation) ApplyForRouteBackend(
	ctx context.Context,
	policy ir.PolicyIR,
	pCtx *ir.RouteBackendContext,
) error {
	return nil
}

func (p *policyTranslation) ApplyForBackend(
	ctx context.Context,
	pCtx *ir.RouteBackendContext,
	in ir.HttpBackend,
	out *routev3.Route,
) error {
	logger := contextutils.LoggerFrom(ctx)
	logger.Info("ApplyForBackend")
	if p == nil || p.usedPool == nil {
		logger.Info("ApplyForBackend p == nil || p.usedPool == nil")
		return nil
	}

	// Add things which require basic EPP backend.
	if out.GetRoute() == nil {
		logger.Info("ApplyForBackend out.GetRoute() == nil")
		out.Action = &routev3.Route_Route{Route: &routev3.RouteAction{}}
	}

	// Create the ext_proc per-route override
	override := &extprocv3.ExtProcPerRoute{
		Override: &extprocv3.ExtProcPerRoute_Overrides{
			Overrides: &extprocv3.ExtProcOverrides{
				GrpcService: &corev3.GrpcService{
					Timeout: durationpb.New(10 * time.Second),
					TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
							ClusterName: clusterNameExtProc(
								p.usedPool.ObjMeta.GetName(),
								p.usedPool.ObjMeta.GetNamespace(),
							),
							Authority: fmt.Sprintf("%s.%s.svc.cluster.local:%d",
								p.usedPool.ConfigRef.Name,
								p.usedPool.ObjMeta.GetNamespace(),
								p.usedPool.ConfigRef.Ports[0].PortNum),
						},
					},
				},
			},
		},
	}

	// Attach to typed_per_filter_config, referencing the same filter name used in the HCM.
	pCtx.AddTypedConfig(wellknown.InfPoolBackendTransformationFilterName, override)

	// Override the route's cluster to point to the ORIGINAL_DST cluster
	originalDstClusterName := clusterNameOriginalDst(p.usedPool.ObjMeta.GetName(), p.usedPool.ObjMeta.GetNamespace())
	out.GetRoute().ClusterSpecifier = &routev3.RouteAction_Cluster{
		Cluster: originalDstClusterName,
	}

	r := out.GetRoute()
	if r == nil {
		logger.Info("ApplyForBackend out.GetRoute()==nil")
	}
	logger.Info("ApplyForBackend out.GetRoute() %v", *r)

	logger.Info("ApplyForBackend return nil")

	return nil
}

// HttpFilters inserts one ext_proc filter at the top-level.
func (p *policyTranslation) HttpFilters(ctx context.Context, fc ir.FilterChainCommon) ([]plugins.StagedHttpFilter, error) {
	logger := contextutils.LoggerFrom(ctx)
	logger.Info("HttpFilters()")
	if p == nil || p.usedPool == nil {
		logger.Info("HttpFilters() p == nil || p.usedPool == nil")
		return nil, nil
	}

	pool := p.usedPool
	clusterName := clusterNameExtProc(pool.ObjMeta.GetName(), pool.ObjMeta.GetNamespace())
	authority := fmt.Sprintf("%s.%s:%d", pool.ConfigRef.Name, pool.ObjMeta.Namespace, pool.ConfigRef.Ports[0].PortNum)

	ret, err := AddEndpointPickerHTTPFilter(clusterName, authority)
	if err != nil {
		logger.Info("HttpFilters() if error != nil %v", err)
	}
	if len(ret) == 0 {
		logger.Info("HttpFilters() len(ret) == 0")
	} else {
		for _, r := range ret {
			logger.Info("HttpFilters() for _, r := range ret %v", r)
		}
	}

	return ret, nil
}

// AddEndpointPickerHTTPFilter returns a top-level ext_proc filter that references
// the cluster built in ResourcesToAdd(). This filter gets placed in the HCM's http_filters array.
func AddEndpointPickerHTTPFilter(clusterName, authority string) ([]plugins.StagedHttpFilter, error) {
	var filters []plugins.StagedHttpFilter

	// This is the top-level ext_proc filter config
	extProcSettings := &extprocv3.ExternalProcessor{
		GrpcService: &corev3.GrpcService{
			TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
				EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
					ClusterName: clusterName,
					Authority:   authority,
				},
			},
		},
		ProcessingMode: &extprocv3.ProcessingMode{
			RequestHeaderMode:   extprocv3.ProcessingMode_SEND,
			RequestBodyMode:     extprocv3.ProcessingMode_BUFFERED,
			ResponseHeaderMode:  extprocv3.ProcessingMode_SKIP,
			RequestTrailerMode:  extprocv3.ProcessingMode_SKIP,
			ResponseTrailerMode: extprocv3.ProcessingMode_SKIP,
		},
		MessageTimeout:   durationpb.New(5 * time.Second),
		FailureModeAllow: false,
	}

	stagedFilter, err := plugins.NewStagedFilter(
		wellknown.InfPoolBackendTransformationFilterName, // Filters must have a unique name.
		extProcSettings,
		plugins.BeforeStage(plugins.RouteStage),
	)
	if err != nil {
		return nil, err
	}
	filters = append(filters, stagedFilter)

	return filters, nil
}

// ResourcesToAdd is called one time (per envoy proxy) and replaces GeneratedResources
// with the returned cluster resources.
func (p *policyTranslation) ResourcesToAdd(ctx context.Context) ir.Resources {
	logger := contextutils.LoggerFrom(ctx)
	logger.Info("ResourcesToAdd()")
	if p == nil || p.usedPool == nil {
		logger.Info("ResourcesToAdd() p == nil || p.usedPool == nil")
		return ir.Resources{}
	}

	// Build an ext-proc cluster for the InferencePool
	cluster := buildExtProcCluster(p.usedPool)
	if cluster == nil {
		logger.Info("ResourcesToAdd() cluster == nil")
	} else {
		logger.Info("ResourcesToAdd() cluster %v", *cluster)
	}
	return ir.Resources{Clusters: []*clusterv3.Cluster{cluster}}
}

// buildExtProcCluster returns a “STRICT_DNS” cluster using the host/port from the given pool.
func buildExtProcCluster(pool *ir.InferencePool) *clusterv3.Cluster {
	name := clusterNameExtProc(pool.ObjMeta.GetName(), pool.ObjMeta.GetNamespace())
	c := &clusterv3.Cluster{
		Name:           name,
		ConnectTimeout: durationpb.New(10 * time.Second),
		ClusterDiscoveryType: &clusterv3.Cluster_Type{
			Type: clusterv3.Cluster_STRICT_DNS,
		},
		LbPolicy: clusterv3.Cluster_LEAST_REQUEST,
		LoadAssignment: &endpointv3.ClusterLoadAssignment{
			ClusterName: name,
			Endpoints: []*endpointv3.LocalityLbEndpoints{{
				LbEndpoints: []*endpointv3.LbEndpoint{{
					HealthStatus: corev3.HealthStatus_HEALTHY,
					HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
						Endpoint: &endpointv3.Endpoint{
							Address: &corev3.Address{
								Address: &corev3.Address_SocketAddress{
									SocketAddress: &corev3.SocketAddress{
										Address: fmt.Sprintf(
											"%s.%s.svc.cluster.local",
											pool.ConfigRef.Name,
											pool.ObjMeta.Namespace,
										),
										Protocol: corev3.SocketAddress_TCP,
										PortSpecifier: &corev3.SocketAddress_PortValue{
											PortValue: uint32(pool.ConfigRef.Ports[0].PortNum),
										},
									},
								},
							},
						},
					},
				}},
			}},
		},
		// Ensure Envoy accepts untrusted certificates.
		TransportSocket: &corev3.TransportSocket{
			Name: "envoy.transport_sockets.tls",
			ConfigType: &corev3.TransportSocket_TypedConfig{
				TypedConfig: func() *anypb.Any {
					tlsCtx := &tlsv3.UpstreamTlsContext{
						CommonTlsContext: &tlsv3.CommonTlsContext{
							ValidationContextType: &tlsv3.CommonTlsContext_ValidationContext{},
						},
					}
					anyTLS, _ := anypb.New(tlsCtx)
					return anyTLS
				}(),
			},
		},
	}

	http2Opts := &upstreamsv3.HttpProtocolOptions{
		UpstreamProtocolOptions: &upstreamsv3.HttpProtocolOptions_ExplicitHttpConfig_{
			ExplicitHttpConfig: &upstreamsv3.HttpProtocolOptions_ExplicitHttpConfig{
				ProtocolConfig: &upstreamsv3.HttpProtocolOptions_ExplicitHttpConfig_Http2ProtocolOptions{
					Http2ProtocolOptions: &corev3.Http2ProtocolOptions{},
				},
			},
		},
	}

	// Marshall the HttpProtocolOptions proto message.
	anyHttp2, _ := utils.MessageToAny(http2Opts)
	c.TypedExtensionProtocolOptions = map[string]*anypb.Any{
		"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": anyHttp2,
	}

	return c
}

func clusterNameExtProc(name, ns string) string {
	return fmt.Sprintf("endpointpicker_%s_%s_ext_proc", name, ns)
}

func clusterNameOriginalDst(name, ns string) string {
	return fmt.Sprintf("endpointpicker_%s_%s_original_dst", name, ns)
}
