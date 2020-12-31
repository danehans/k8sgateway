// Code generated by protoc-gen-ext. DO NOT EDIT.
// source: github.com/solo-io/gloo/projects/gloo/api/external/envoy/config/trace/v3/datadog.proto

package v3

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	equality "github.com/solo-io/protoc-gen-ext/pkg/equality"
)

// ensure the imports are used
var (
	_ = errors.New("")
	_ = fmt.Print
	_ = binary.LittleEndian
	_ = bytes.Compare
	_ = strings.Compare
	_ = equality.Equalizer(nil)
	_ = proto.Message(nil)
)

// Equal function
func (m *DatadogConfig) Equal(that interface{}) bool {
	if that == nil {
		return m == nil
	}

	target, ok := that.(*DatadogConfig)
	if !ok {
		that2, ok := that.(DatadogConfig)
		if ok {
			target = &that2
		} else {
			return false
		}
	}
	if target == nil {
		return m == nil
	} else if m == nil {
		return false
	}

	if strings.Compare(m.GetServiceName(), target.GetServiceName()) != 0 {
		return false
	}

	switch m.CollectorCluster.(type) {

	case *DatadogConfig_CollectorUpstreamRef:

		if h, ok := interface{}(m.GetCollectorUpstreamRef()).(equality.Equalizer); ok {
			if !h.Equal(target.GetCollectorUpstreamRef()) {
				return false
			}
		} else {
			if !proto.Equal(m.GetCollectorUpstreamRef(), target.GetCollectorUpstreamRef()) {
				return false
			}
		}

	case *DatadogConfig_ClusterName:

		if strings.Compare(m.GetClusterName(), target.GetClusterName()) != 0 {
			return false
		}

	}

	return true
}