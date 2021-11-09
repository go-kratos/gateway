// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.17.3
// source: gateway/middleware/prometheus/prometheus.proto

package v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Prometheus middleware config
type Prometheus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Path string `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
}

func (x *Prometheus) Reset() {
	*x = Prometheus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_gateway_middleware_prometheus_prometheus_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Prometheus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Prometheus) ProtoMessage() {}

func (x *Prometheus) ProtoReflect() protoreflect.Message {
	mi := &file_gateway_middleware_prometheus_prometheus_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Prometheus.ProtoReflect.Descriptor instead.
func (*Prometheus) Descriptor() ([]byte, []int) {
	return file_gateway_middleware_prometheus_prometheus_proto_rawDescGZIP(), []int{0}
}

func (x *Prometheus) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

var File_gateway_middleware_prometheus_prometheus_proto protoreflect.FileDescriptor

var file_gateway_middleware_prometheus_prometheus_proto_rawDesc = []byte{
	0x0a, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2f, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65,
	0x77, 0x61, 0x72, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x65, 0x75, 0x73, 0x2f,
	0x70, 0x72, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x65, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x20, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65,
	0x77, 0x61, 0x72, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x65, 0x75, 0x73, 0x2e,
	0x76, 0x31, 0x22, 0x20, 0x0a, 0x0a, 0x50, 0x72, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x65, 0x75, 0x73,
	0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x70, 0x61, 0x74, 0x68, 0x42, 0x43, 0x5a, 0x41, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x2d, 0x6b, 0x72, 0x61, 0x74, 0x6f, 0x73, 0x2f, 0x67, 0x61, 0x74,
	0x65, 0x77, 0x61, 0x79, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79,
	0x2f, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x6d,
	0x65, 0x74, 0x68, 0x65, 0x75, 0x73, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_gateway_middleware_prometheus_prometheus_proto_rawDescOnce sync.Once
	file_gateway_middleware_prometheus_prometheus_proto_rawDescData = file_gateway_middleware_prometheus_prometheus_proto_rawDesc
)

func file_gateway_middleware_prometheus_prometheus_proto_rawDescGZIP() []byte {
	file_gateway_middleware_prometheus_prometheus_proto_rawDescOnce.Do(func() {
		file_gateway_middleware_prometheus_prometheus_proto_rawDescData = protoimpl.X.CompressGZIP(file_gateway_middleware_prometheus_prometheus_proto_rawDescData)
	})
	return file_gateway_middleware_prometheus_prometheus_proto_rawDescData
}

var file_gateway_middleware_prometheus_prometheus_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_gateway_middleware_prometheus_prometheus_proto_goTypes = []interface{}{
	(*Prometheus)(nil), // 0: gateway.middleware.prometheus.v1.Prometheus
}
var file_gateway_middleware_prometheus_prometheus_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_gateway_middleware_prometheus_prometheus_proto_init() }
func file_gateway_middleware_prometheus_prometheus_proto_init() {
	if File_gateway_middleware_prometheus_prometheus_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_gateway_middleware_prometheus_prometheus_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Prometheus); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_gateway_middleware_prometheus_prometheus_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_gateway_middleware_prometheus_prometheus_proto_goTypes,
		DependencyIndexes: file_gateway_middleware_prometheus_prometheus_proto_depIdxs,
		MessageInfos:      file_gateway_middleware_prometheus_prometheus_proto_msgTypes,
	}.Build()
	File_gateway_middleware_prometheus_prometheus_proto = out.File
	file_gateway_middleware_prometheus_prometheus_proto_rawDesc = nil
	file_gateway_middleware_prometheus_prometheus_proto_goTypes = nil
	file_gateway_middleware_prometheus_prometheus_proto_depIdxs = nil
}
