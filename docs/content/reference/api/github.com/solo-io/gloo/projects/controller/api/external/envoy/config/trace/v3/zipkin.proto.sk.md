
---
title: "zipkin.proto"
weight: 5
---

<!-- Code generated by solo-kit. DO NOT EDIT. -->


### Package: `solo.io.envoy.config.trace.v3` 
#### Types:


- [ZipkinConfig](#zipkinconfig)
- [CollectorEndpointVersion](#collectorendpointversion)
  



##### Source File: [github.com/solo-io/gloo/projects/controller/api/external/envoy/config/trace/v3/zipkin.proto](https://github.com/solo-io/gloo/blob/main/projects/controller/api/external/envoy/config/trace/v3/zipkin.proto)





---
### ZipkinConfig

 
Configuration for the Zipkin tracer.
[#extension: envoy.tracers.zipkin]
[#next-free-field: 6]

```yaml
"collectorUpstreamRef": .core.solo.io.ResourceRef
"clusterName": string
"collectorEndpoint": string
"traceId128Bit": .google.protobuf.BoolValue
"sharedSpanContext": .google.protobuf.BoolValue
"collectorEndpointVersion": .solo.io.envoy.config.trace.v3.ZipkinConfig.CollectorEndpointVersion

```

| Field | Type | Description |
| ----- | ---- | ----------- | 
| `collectorUpstreamRef` | [.core.solo.io.ResourceRef](../../../../../../../../../../solo-kit/api/v1/ref.proto.sk/#resourceref) | The upstream that hosts the Zipkin collectors. Only one of `collectorUpstreamRef` or `clusterName` can be set. |
| `clusterName` | `string` | The name of the cluster that hosts the Zipkin collectors. Note that the Zipkin cluster must be defined in the :ref:`Bootstrap static cluster resources <envoy_api_field_config.bootstrap.v3.Bootstrap.StaticResources.clusters>`. Only one of `clusterName` or `collectorUpstreamRef` can be set. |
| `collectorEndpoint` | `string` | The API endpoint of the Zipkin service where the spans will be sent. When using a standard Zipkin installation, the API endpoint is typically /api/v1/spans, which is the default value. |
| `traceId128Bit` | [.google.protobuf.BoolValue](https://developers.google.com/protocol-buffers/docs/reference/csharp/class/google/protobuf/well-known-types/bool-value) | Determines whether a 128bit trace id will be used when creating a new trace instance. The default value is false, which will result in a 64 bit trace id being used. |
| `sharedSpanContext` | [.google.protobuf.BoolValue](https://developers.google.com/protocol-buffers/docs/reference/csharp/class/google/protobuf/well-known-types/bool-value) | Determines whether client and server spans will share the same span context. The default value is true. |
| `collectorEndpointVersion` | [.solo.io.envoy.config.trace.v3.ZipkinConfig.CollectorEndpointVersion](../zipkin.proto.sk/#collectorendpointversion) | Determines the selected collector endpoint version. By default, the `HTTP_JSON_V1` will be used. |




---
### CollectorEndpointVersion

 
Available Zipkin collector endpoint versions.

| Name | Description |
| ----- | ----------- | 
| `DEPRECATED_AND_UNAVAILABLE_DO_NOT_USE` | Zipkin API v1, JSON over HTTP. [#comment: The default implementation of Zipkin client before this field is added was only v1 and the way user configure this was by not explicitly specifying the version. Consequently, before this is added, the corresponding Zipkin collector expected to receive v1 payload. Hence the motivation of adding HTTP_JSON_V1 as the default is to avoid a breaking change when user upgrading Envoy with this change. Furthermore, we also immediately deprecate this field, since in Zipkin realm this v1 version is considered to be not preferable anymore.] |
| `HTTP_JSON` | Zipkin API v2, JSON over HTTP. |
| `HTTP_PROTO` | Zipkin API v2, protobuf over HTTP. |





<!-- Start of HubSpot Embed Code -->
<script type="text/javascript" id="hs-script-loader" async defer src="//js.hs-scripts.com/5130874.js"></script>
<!-- End of HubSpot Embed Code -->