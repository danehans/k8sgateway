
---
title: "cidr.proto"
weight: 5
---

<!-- Code generated by solo-kit. DO NOT EDIT. -->


### Package: `xds.core.v3` 
#### Types:


- [CidrRange](#cidrrange)
  



##### Source File: [github.com/solo-io/gloo/projects/controller/api/external/xds/core/v3/cidr.proto](https://github.com/solo-io/gloo/blob/main/projects/controller/api/external/xds/core/v3/cidr.proto)





---
### CidrRange

 
CidrRange specifies an IP Address and a prefix length to construct
the subnet mask for a [CIDR](https://datatracker.ietf.org/doc/html/rfc4632) range.

```yaml
"addressPrefix": string
"prefixLen": .google.protobuf.UInt32Value

```

| Field | Type | Description |
| ----- | ---- | ----------- | 
| `addressPrefix` | `string` | IPv4 or IPv6 address, e.g. `192.0.0.0` or `2001:db8::`. |
| `prefixLen` | [.google.protobuf.UInt32Value](https://developers.google.com/protocol-buffers/docs/reference/csharp/class/google/protobuf/well-known-types/u-int-32-value) | Length of prefix, e.g. 0, 32. Defaults to 0 when unset. |





<!-- Start of HubSpot Embed Code -->
<script type="text/javascript" id="hs-script-loader" async defer src="//js.hs-scripts.com/5130874.js"></script>
<!-- End of HubSpot Embed Code -->