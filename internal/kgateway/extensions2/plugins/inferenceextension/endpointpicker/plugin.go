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
	ext_procv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ext_proc/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	httpv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"github.com/solo-io/go-utils/contextutils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"istio.io/istio/pkg/kube/kclient"
	"istio.io/istio/pkg/kube/krt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	infextv1a1 "sigs.k8s.io/gateway-api-inference-extension/api/v1alpha1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/extensions2/common"
	extplug "github.com/kgateway-dev/kgateway/v2/internal/kgateway/extensions2/plugin"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/extensions2/settings"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/ir"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/krtcollections"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/plugins"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/utils"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/utils/krtutil"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/wellknown"
)

// TODO [danehans]: Filter InferencePools based one's that are referenced by an HTTPRoute
// with a status.parents[].controllerName that matches our Gateway controllerName.

func NewPlugin(ctx context.Context, commonCol *common.CommonCollections) extplug.Plugin {
	poolClient := kclient.New[*infextv1a1.InferencePool](commonCol.Client)
	pools := krt.WrapClient(poolClient, commonCol.KrtOpts.ToOptions("InferencePools")...)
	routeClient := kclient.New[*gwv1.HTTPRoute](commonCol.Client)
	routes := krt.WrapClient(routeClient, commonCol.KrtOpts.ToOptions("HTTPRoutes")...)
	svcClient := kclient.New[*corev1.Service](commonCol.Client)
	svcs := krt.WrapClient(svcClient, commonCol.KrtOpts.ToOptions("Services")...)
	return NewPluginFromCollections(ctx, commonCol, pools, routes, svcs, commonCol.Pods, commonCol.Settings)
}

func NewPluginFromCollections(
	ctx context.Context,
	commonCol *common.CommonCollections,
	poolCol krt.Collection[*infextv1a1.InferencePool],
	routeCol krt.Collection[*gwv1.HTTPRoute],
	svcCol krt.Collection[*corev1.Service],
	podCol krt.Collection[krtcollections.LocalityPod],
	stngs settings.Settings,
) extplug.Plugin {
	// Create an index on HTTPRoutes by the InferencePool they reference.
	httpRoutesByInferencePool := krt.NewIndex(routeCol, func(route *gwv1.HTTPRoute) []types.NamespacedName {
		var refs []types.NamespacedName
		for _, rule := range route.Spec.Rules {
			for _, backend := range rule.BackendRefs {
				if backend.Kind != nil && *backend.Kind == wellknown.InferencePoolKind {
					refs = append(refs, types.NamespacedName{
						Namespace: route.Namespace,
						Name:      string(backend.Name),
					})
				}
			}
		}
		return refs
	})

	// The InferencePool group kind used by the BackendObjectIR and the ContributesBackendObjectIRs plugin.
	gk := schema.GroupKind{
		Group: infextv1a1.GroupVersion.Group,
		Kind:  wellknown.InferencePoolKind,
	}

	// Build the Service IR from the Service collection.
	irSvcs := buildServiceIRCollection(svcCol, commonCol)

	// Build the BackendObjectIR translation function from the Service IR.
	translate := buildTranslateFunc(ctx, krtcollections.NewServiceIndex(irSvcs))

	// Create a BackendObjectIR from the InferencePool.
	us := krt.NewCollection(poolCol, func(kctx krt.HandlerContext, pool *infextv1a1.InferencePool) *ir.BackendObjectIR {
		poolKey := types.NamespacedName{Namespace: pool.Namespace, Name: pool.Name}
		matchingRoutes := httpRoutesByInferencePool.Lookup(poolKey)

		valid := false
		for _, route := range matchingRoutes {
			// Iterate over status.parents and check if any match the Gateway controller
			for _, parent := range route.Status.Parents {
				if parent.ControllerName == gwv1.GatewayController(wellknown.GatewayControllerName) {
					valid = true
					break
				}
			}
			if valid {
				// Only one match is required for an InferencePool to be considered managed by this plugin.
				break
			}
		}

		if !valid {
			// Skip this InferencePool if it has no valid HTTPRoute references.
			// TODO [danehans]: Surface a status condition.
			return nil
		}

		// This InferencePool is valid, create an BackendObjectIR IR representation.
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
			ObjIr:             translate(kctx, pool),
		}
	}, commonCol.KrtOpts.ToOptions("EndpointPickerBackendObjectIR")...)

	// Create an endpoints krt collection from the BackendObjectIR.
	inputs := newInfPoolEndpointsInputs(commonCol.KrtOpts, us, podCol)
	infPoolEndpoints := newInfPoolEndpoints(ctx, inputs)

	return extplug.Plugin{
		ContributesBackends: map[schema.GroupKind]extplug.BackendPlugin{
			gk: {
				BackendInit: ir.BackendInit{
					InitBackend: processBackendObjectIR,
				},
				Endpoints: infPoolEndpoints,
				Backends:  us,
			},
		},
		ContributesPolicies: map[schema.GroupKind]extplug.PolicyPlugin{
			gk: {
				Name: "endpointpicker-extproc",
				NewGatewayTranslationPass: func(ctx context.Context, tctx ir.GwTranslationCtx) ir.ProxyTranslationPass {
					return newEndpointPickerPass(us, infPoolEndpoints)
				},
			},
		},
	}
}

func buildServiceIRCollection(
	svcCol krt.Collection[*corev1.Service],
	commonCol *common.CommonCollections,
) map[schema.GroupKind]krt.Collection[ir.Service] {
	// Create a krt collection for the endpoint picker extension Service and translate to the Service IR.
	svcIR := krt.NewCollection(svcCol, func(kctx krt.HandlerContext, svc *corev1.Service) *ir.Service {
		ports := []ir.ServicePort{}
		for _, p := range svc.Spec.Ports {
			irPort := ir.ServicePort{
				PortNum: p.Port,
			}
			switch {
			case p.Name != "":
				irPort.Name = p.Name
			case p.Port == int32(ir.GRPCPort):
				irPort.Name = "grpc"
			}
			ports = append(ports, ir.ServicePort{PortNum: p.Port})
		}

		return &ir.Service{
			ObjectSource: ir.ObjectSource{
				Group:     corev1.SchemeGroupVersion.Group,
				Kind:      wellknown.ServiceKind,
				Namespace: svc.Namespace,
				Name:      svc.Name,
			},
			Obj:   svc,
			Ports: ports,
		}
	}, commonCol.KrtOpts.ToOptions("ServiceIR")...)

	return map[schema.GroupKind]krt.Collection[ir.Service]{
		{Group: "", Kind: "Service"}: svcIR,
	}
}

func buildTranslateFunc(ctx context.Context, svcs *krtcollections.ServiceIndex) func(kctx krt.HandlerContext, pool *infextv1a1.InferencePool) *ir.InferencePool {
	return func(kctx krt.HandlerContext, pool *infextv1a1.InferencePool) *ir.InferencePool {
		var ret ir.InferencePool

		if pool.Spec.Selector == nil {
			ret.PodSelector = make(map[string]string)
		} else {
			ret.PodSelector = convertSelector(pool.Spec.Selector)
		}

		ret.TargetPort = pool.Spec.TargetPortNumber

		svc := svcs.GetSvcForInferPool(kctx, pool)
		if svc != nil {
			ret.ConfigRef = svc
		} else {
			// TODO [danehans]: Log and write to InferencePool status
		}
		return &ret
	}
}

func processBackendObjectIR(ctx context.Context, in ir.BackendObjectIR, out *clusterv3.Cluster) {
	// Set cluster type to ORIGINAL_DST
	out.ClusterDiscoveryType = &clusterv3.Cluster_Type{
		Type: clusterv3.Cluster_ORIGINAL_DST,
	}

	// Set connect timeout to 1000 seconds.
	// TODO [danehans]: Figure out an API that can be used to set this value.
	out.ConnectTimeout = durationpb.New(1000 * time.Second)

	// Use CLUSTER_PROVIDED load balancing.
	out.LbPolicy = clusterv3.Cluster_CLUSTER_PROVIDED

	// Configure circuit breakers with a single threshold.
	// TODO [danehans]: Figure out an API that can be used to set these values.
	out.CircuitBreakers = &clusterv3.CircuitBreakers{
		Thresholds: []*clusterv3.CircuitBreakers_Thresholds{
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
	lbConfig := &clusterv3.Cluster_OriginalDstLbConfig{
		UseHttpHeader:  true,
		HttpHeaderName: "x-gateway-destination-endpoint",
	}
	anyLbConfig, err := utils.MessageToAny(lbConfig)
	if err != nil {
		// handle error appropriately
		return
	}
	out.TypedExtensionProtocolOptions = map[string]*anypb.Any{
		"envoy.lb": anyLbConfig,
	}
}

type InfPoolEndpointsInputs struct {
	BackendObjectIRs krt.Collection[ir.BackendObjectIR]
	Pods             krt.Collection[krtcollections.LocalityPod]
	KrtOpts          krtutil.KrtOptions
}

func newInfPoolEndpointsInputs(
	krtOpts krtutil.KrtOptions,
	infPoolBackendObjectIRs krt.Collection[ir.BackendObjectIR],
	podCol krt.Collection[krtcollections.LocalityPod],
) InfPoolEndpointsInputs {
	return InfPoolEndpointsInputs{
		BackendObjectIRs: infPoolBackendObjectIRs,
		Pods:             podCol,
		KrtOpts:          krtOpts,
	}
}

func newInfPoolEndpoints(ctx context.Context, inputs InfPoolEndpointsInputs) krt.Collection[ir.EndpointsForBackend] {
	return krt.NewCollection(inputs.BackendObjectIRs, transformInfPoolEndpoints(ctx, inputs), inputs.KrtOpts.ToOptions("InfPoolEndpoints")...)
}

func transformInfPoolEndpoints(ctx context.Context, inputs InfPoolEndpointsInputs) func(kctx krt.HandlerContext, us ir.BackendObjectIR) *ir.EndpointsForBackend {
	logger := contextutils.LoggerFrom(ctx).Desugar()

	return func(kctx krt.HandlerContext, us ir.BackendObjectIR) *ir.EndpointsForBackend {
		infPool, ok := us.Obj.(*infextv1a1.InferencePool)
		if !ok {
			logger.Debug("not an InferencePool object")
			return nil
		}

		logger.Debug("building endpoints for inference pool", zap.String("pool", infPool.Name))

		// Convert `spec.selector` from custom type to `map[string]string`
		labelSelector := convertSelector(infPool.Spec.Selector)

		// Use `FilterGeneric()` to match `LocalityPod` based on AugmentedLabels and Namespace
		podMatches := krt.Fetch(kctx, inputs.Pods, krt.FilterGeneric(func(obj any) bool {
			pod, ok := obj.(krtcollections.LocalityPod)
			if !ok {
				return false
			}
			// Ensure Pod is in the same namespace as the InferencePool
			if pod.Namespace != infPool.Namespace {
				return false
			}
			// Ensure the pod labels match the InferencePool selector
			return labelsMatch(labelSelector, pod.AugmentedLabels)
		}))

		// Always return a valid EndpointsForBackendObjectIR instance, even if no matching pods
		ret := ir.NewEndpointsForBackend(us)

		if len(podMatches) == 0 {
			logger.Debug("no matching pods found for inference pool", zap.String("pool", infPool.Name))
			return ret // Return an empty but valid EndpointsForBackendObjectIR
		}

		// Deduplicate Pod IPs
		seenAddresses := make(map[string]struct{})

		// Process matching Pods
		for _, pod := range podMatches {
			// Get the primary pod address
			podIP := pod.IP()
			if podIP == "" {
				continue
			}

			// Deduplicate addresses
			if _, exists := seenAddresses[podIP]; exists {
				continue
			}
			seenAddresses[podIP] = struct{}{}

			// Create Envoy LB Endpoint
			ep := krtcollections.CreateLBEndpoint(podIP, uint32(infPool.Spec.TargetPortNumber), pod.AugmentedLabels, true)

			// Add endpoint
			ret.Add(pod.Locality, ir.EndpointWithMd{
				LbEndpoint: ep,
				EndpointMd: ir.EndpointMetadata{
					Labels: pod.AugmentedLabels,
				},
			})
		}

		logger.Debug("created endpoints", zap.Int("numAddresses", len(ret.LbEps)))
		return ret
	}
}

func convertSelector(selector map[infextv1a1.LabelKey]infextv1a1.LabelValue) map[string]string {
	result := make(map[string]string, len(selector))
	for k, v := range selector {
		result[string(k)] = string(v)
	}
	return result
}

func labelsMatch(selector map[string]string, podLabels map[string]string) bool {
	for k, v := range selector {
		if podLabels[k] != v {
			return false
		}
	}
	return true
}

// endpointPickerPass implements ir.ProxyTranslationPass.
type endpointPickerPass struct {
	// extProcClusters maps an InferencePool keyed by namespace/name to cluster name.
	// TODO [danehans]: Use typesNamespacedName for key.
	extProcClusters  map[types.NamespacedName]string
	infPoolBackend   krt.Collection[ir.BackendObjectIR]
	infPoolEndpoints krt.Collection[ir.EndpointsForBackend]
}

// newEndpointPickerPass initializes a new endpoint picker instance.
func newEndpointPickerPass(
	infPoolBackendObjectIR krt.Collection[ir.BackendObjectIR],
	infPoolEndpoints krt.Collection[ir.EndpointsForBackend],
) ir.ProxyTranslationPass {
	return &endpointPickerPass{
		extProcClusters:  make(map[types.NamespacedName]string),
		infPoolBackend:   infPoolBackendObjectIR,
		infPoolEndpoints: infPoolEndpoints,
	}
}

// Name identifies this pass.
func (e *endpointPickerPass) Name() string {
	return "endpointpicker-extproc"
}

// ApplyListenerPlugin is invoked once for each Envoy listener. No-op for ext_proc.
func (e *endpointPickerPass) ApplyListenerPlugin(
	ctx context.Context,
	pCtx *ir.ListenerContext,
	out *listenerv3.Listener,
) {
	// no-op
}

// ApplyHCM is invoked once for each HttpConnectionManager config. No-op for ext_proc.
func (e *endpointPickerPass) ApplyHCM(
	ctx context.Context,
	pCtx *ir.HcmContext,
	out *hcmv3.HttpConnectionManager,
) error {
	return nil
}

// ApplyVhostPlugin is invoked for each virtual host. No-op for ext_proc.
func (e *endpointPickerPass) ApplyVhostPlugin(
	ctx context.Context,
	pCtx *ir.VirtualHostContext,
	out *routev3.VirtualHost,
) {
}

// ApplyForRoute is invoked once for each route. No-op for ext_proc here.
func (e *endpointPickerPass) ApplyForRoute(
	ctx context.Context,
	pCtx *ir.RouteContext,
	outputRoute *routev3.Route,
) error {
	return nil
}

// ApplyForRouteBackend is invoked for each backend on each route by detecting
// if the backend references an InferencePool and store ext_proc cluster info.
func (e *endpointPickerPass) ApplyForRouteBackend(
	ctx context.Context,
	policy ir.PolicyIR,
	pCtx *ir.RouteBackendContext,
) error {
	// Check if the backend is InferencePool
	if pCtx.Upstream.Kind != wellknown.InferencePoolKind &&
		pCtx.Upstream.Group != infextv1a1.GroupVersion.Group {
		return fmt.Errorf("unsupported group or kind: must be group %s and kind %s",
			infextv1a1.GroupVersion.Group, wellknown.InferencePoolKind)
	}

	// Cast the object to an InferencePool
	pool, ok := pCtx.Upstream.Obj.(*infextv1a1.InferencePool)
	if !ok || pool == nil {
		return fmt.Errorf("inference pool %s/%s not found", pCtx.Upstream.Namespace, pCtx.Upstream.Name)
	}

	// Validate the InferencePool extension reference
	ref := pool.Spec.ExtensionRef
	if ref == nil {
		return fmt.Errorf("inference pool %s/%s missing extensionRef", pool.Namespace, pool.Name)
	}
	if (ref.Kind != nil && *ref.Kind != wellknown.ServiceKind) || (ref.Group != nil && *ref.Group != "") {
		return fmt.Errorf(
			"invalid extensionRef for inference pool %s/%s", pool.Namespace, pool.Name)
	}

	// Build a unique name for the ext_proc cluster, e.g. ext-proc-<namespace>-<poolName>.
	clusterName := fmt.Sprintf("ext-proc-%s-%s", pool.Namespace, pool.Name)
	e.extProcClusters[types.NamespacedName{Namespace: pool.Namespace, Name: pool.Name}] = clusterName

	// Optionally, set typed_per_filter_config on this route or return that config
	// so the ext_proc filter references the cluster.
	override := &ext_procv3.ExtProcPerRoute{
		Override: &ext_procv3.ExtProcPerRoute_Overrides{
			Overrides: &ext_procv3.ExtProcOverrides{
				GrpcService: &corev3.GrpcService{
					TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
							ClusterName: clusterName,
						},
					},
				},
			},
		},
	}
	anyOverride, err := utils.MessageToAny(override)
	if err != nil {
		return fmt.Errorf("failed to marshal ext_proc per-route override: %w", err)
	}

	// Attach the typed_per_filter_config to the route backend context.
	pCtx.AddTypedConfig("envoy.filters.http.ext_proc", anyOverride)

	return nil
}

// HttpFilters is called once per filter chain. If extProcNeeded, we add the ext_proc filter.
func (e *endpointPickerPass) HttpFilters(
	ctx context.Context,
	fc ir.FilterChainCommon,
) ([]plugins.StagedHttpFilter, error) {
	// Build the ExternalProcessor config (without a default gRPC service, since it's set per route).
	extProc := &ext_procv3.ExternalProcessor{
		// TODO [danehans]: Failure mode should be set based on InferencePool extensionRef failureMode.
		FailureModeAllow: false,
		ProcessingMode: &ext_procv3.ProcessingMode{
			RequestHeaderMode:  ext_procv3.ProcessingMode_SEND,
			ResponseHeaderMode: ext_procv3.ProcessingMode_SKIP,
			RequestBodyMode:    ext_procv3.ProcessingMode_BUFFERED,
			ResponseBodyMode:   ext_procv3.ProcessingMode_NONE,
		},
	}
	anyExtProc, err := utils.MessageToAny(extProc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ext_proc filter config: %w", err)
	}

	// Assign the ext_proc filter to the pre-routing stage.
	stagedFilter, err := plugins.NewStagedFilter(
		"envoy.filters.http.ext_proc",
		anyExtProc,
		plugins.BeforeStage(plugins.RouteStage),
	)
	if err != nil {
		return nil, err
	}
	return []plugins.StagedHttpFilter{stagedFilter}, nil
}

// UpstreamHttpFilters: no upstream-level filters needed for ext_proc, so return nil.
func (e *endpointPickerPass) UpstreamHttpFilters(ctx context.Context) ([]plugins.StagedUpstreamHttpFilter, error) {
	return nil, nil
}

// NetworkFilters: no network-level filters for ext_proc, so return nil.
func (e *endpointPickerPass) NetworkFilters(ctx context.Context) ([]plugins.StagedNetworkFilter, error) {
	return nil, nil
}

// ResourcesToAdd is called once to let this pass add new Envoy resources, e.g. clusters.
func (e *endpointPickerPass) ResourcesToAdd(ctx context.Context) ir.Resources {
	var result ir.Resources

	for key, clusterName := range e.extProcClusters {
		var kctx krt.HandlerContext
		backend := krt.FetchOne(kctx, e.infPoolBackend, krt.FilterObjectName(key))
		if backend == nil {
			continue
		}
		pool, ok := backend.Obj.(*infextv1a1.InferencePool)
		if !ok {
			continue
		}
		ref := pool.Spec.ExtensionRef
		if ref == nil {
			// Shouldn't happen if we validated above
			continue
		}
		port := int32(9002)
		if ref.TargetPortNumber != nil && *ref.TargetPortNumber > 0 {
			port = *ref.TargetPortNumber
		}

		// Build the ext_proc cluster
		svcHost := fmt.Sprintf("%s.%s.svc.cluster.local", ref.Name, key.Namespace)
		extProcCluster := &clusterv3.Cluster{
			Name:                 clusterName,
			ConnectTimeout:       durationpb.New(24 * time.Hour), // 86400s
			ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_STRICT_DNS},
			LbPolicy:             clusterv3.Cluster_LEAST_REQUEST,
			CircuitBreakers: &clusterv3.CircuitBreakers{
				Thresholds: []*clusterv3.CircuitBreakers_Thresholds{{
					MaxConnections:     wrapperspb.UInt32(40000),
					MaxPendingRequests: wrapperspb.UInt32(40000),
					MaxRequests:        wrapperspb.UInt32(40000),
					MaxRetries:         wrapperspb.UInt32(1024),
				}},
			},
			LoadAssignment: &endpointv3.ClusterLoadAssignment{
				ClusterName: clusterName,
				Endpoints: []*endpointv3.LocalityLbEndpoints{{
					Locality: &corev3.Locality{
						// TODO [danehans]: Get this value from ir.PodLocality of extension service pods?
						Region: "ext_proc",
					},
					LbEndpoints: []*endpointv3.LbEndpoint{{
						HealthStatus:        corev3.HealthStatus_HEALTHY,
						LoadBalancingWeight: &wrapperspb.UInt32Value{Value: 1},
						HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
							Endpoint: &endpointv3.Endpoint{
								Address: &corev3.Address{
									Address: &corev3.Address_SocketAddress{
										SocketAddress: &corev3.SocketAddress{
											Address: svcHost,
											PortSpecifier: &corev3.SocketAddress_PortValue{
												PortValue: uint32(port),
											},
											Protocol: corev3.SocketAddress_TCP,
										},
									},
								},
							},
						},
					}},
				}},
			},
			// Accept untrusted certs by leaving the validation context empty
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
		// Enable HTTP/2. We attach typed extension protocol options for http2, with big window sizes
		http2Opts := &httpv3.HttpProtocolOptions{
			UpstreamProtocolOptions: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_{
				ExplicitHttpConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig{
					ProtocolConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_Http2ProtocolOptions{
						Http2ProtocolOptions: &corev3.Http2ProtocolOptions{
							MaxConcurrentStreams:        wrapperspb.UInt32(100),
							InitialStreamWindowSize:     wrapperspb.UInt32(65536),
							InitialConnectionWindowSize: wrapperspb.UInt32(1048576),
						},
					},
				},
			},
		}
		anyHTTP2, _ := anypb.New(http2Opts)
		extProcCluster.TypedExtensionProtocolOptions = map[string]*anypb.Any{
			"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": anyHTTP2,
		}

		// Add the cluster to our resources
		result.Clusters = append(result.Clusters, extProcCluster)
	}

	return result
}
