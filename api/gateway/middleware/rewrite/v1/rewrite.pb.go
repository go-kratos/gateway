// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.20.0
// source: gateway/middleware/rewrite/v1/rewrite.proto

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

// Rewrite middleware config.
type HeadersPolicy struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Set    map[string]string `protobuf:"bytes,1,rep,name=set,proto3" json:"set,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Add    map[string]string `protobuf:"bytes,2,rep,name=add,proto3" json:"add,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Remove []string          `protobuf:"bytes,3,rep,name=remove,proto3" json:"remove,omitempty"`
}

func (x *HeadersPolicy) Reset() {
	*x = HeadersPolicy{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_gateway_middleware_rewrite_v1_rewrite_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HeadersPolicy) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HeadersPolicy) ProtoMessage() {}

func (x *HeadersPolicy) ProtoReflect() protoreflect.Message {
	mi := &file_api_gateway_middleware_rewrite_v1_rewrite_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HeadersPolicy.ProtoReflect.Descriptor instead.
func (*HeadersPolicy) Descriptor() ([]byte, []int) {
	return file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDescGZIP(), []int{0}
}

func (x *HeadersPolicy) GetSet() map[string]string {
	if x != nil {
		return x.Set
	}
	return nil
}

func (x *HeadersPolicy) GetAdd() map[string]string {
	if x != nil {
		return x.Add
	}
	return nil
}

func (x *HeadersPolicy) GetRemove() []string {
	if x != nil {
		return x.Remove
	}
	return nil
}

type Rewrite struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	PathRewrite            *string        `protobuf:"bytes,1,opt,name=path_rewrite,json=pathRewrite,proto3,oneof" json:"path_rewrite,omitempty"`
	RequestHeadersRewrite  *HeadersPolicy `protobuf:"bytes,2,opt,name=request_headers_rewrite,json=requestHeadersRewrite,proto3" json:"request_headers_rewrite,omitempty"`
	ResponseHeadersRewrite *HeadersPolicy `protobuf:"bytes,3,opt,name=response_headers_rewrite,json=responseHeadersRewrite,proto3" json:"response_headers_rewrite,omitempty"`
	StripPrefix            *int64         `protobuf:"varint,4,opt,name=stripPrefix,proto3,oneof" json:"stripPrefix,omitempty"`
}

func (x *Rewrite) Reset() {
	*x = Rewrite{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_gateway_middleware_rewrite_v1_rewrite_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Rewrite) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Rewrite) ProtoMessage() {}

func (x *Rewrite) ProtoReflect() protoreflect.Message {
	mi := &file_api_gateway_middleware_rewrite_v1_rewrite_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Rewrite.ProtoReflect.Descriptor instead.
func (*Rewrite) Descriptor() ([]byte, []int) {
	return file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDescGZIP(), []int{1}
}

func (x *Rewrite) GetPathRewrite() string {
	if x != nil && x.PathRewrite != nil {
		return *x.PathRewrite
	}
	return ""
}

func (x *Rewrite) GetRequestHeadersRewrite() *HeadersPolicy {
	if x != nil {
		return x.RequestHeadersRewrite
	}
	return nil
}

func (x *Rewrite) GetResponseHeadersRewrite() *HeadersPolicy {
	if x != nil {
		return x.ResponseHeadersRewrite
	}
	return nil
}

func (x *Rewrite) GetStripPrefix() int64 {
	if x != nil && x.StripPrefix != nil {
		return *x.StripPrefix
	}
	return 0
}

var File_api_gateway_middleware_rewrite_v1_rewrite_proto protoreflect.FileDescriptor

var file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDesc = []byte{
	0x0a, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2f, 0x6d, 0x69,
	0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2f, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65,
	0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x1d, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x6d, 0x69, 0x64, 0x64, 0x6c,
	0x65, 0x77, 0x61, 0x72, 0x65, 0x2e, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x2e, 0x76, 0x31,
	0x22, 0xa9, 0x02, 0x0a, 0x0d, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x50, 0x6f, 0x6c, 0x69,
	0x63, 0x79, 0x12, 0x47, 0x0a, 0x03, 0x73, 0x65, 0x74, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x35, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65,
	0x77, 0x61, 0x72, 0x65, 0x2e, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e,
	0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x2e, 0x53, 0x65,
	0x74, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x03, 0x73, 0x65, 0x74, 0x12, 0x47, 0x0a, 0x03, 0x61,
	0x64, 0x64, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x35, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77,
	0x61, 0x79, 0x2e, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2e, 0x72, 0x65,
	0x77, 0x72, 0x69, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73,
	0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x2e, 0x41, 0x64, 0x64, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52,
	0x03, 0x61, 0x64, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x18, 0x03,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x1a, 0x36, 0x0a, 0x08,
	0x53, 0x65, 0x74, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x3a, 0x02, 0x38, 0x01, 0x1a, 0x36, 0x0a, 0x08, 0x41, 0x64, 0x64, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xc7, 0x02, 0x0a,
	0x07, 0x52, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x12, 0x26, 0x0a, 0x0c, 0x70, 0x61, 0x74, 0x68,
	0x5f, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00,
	0x52, 0x0b, 0x70, 0x61, 0x74, 0x68, 0x52, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x88, 0x01, 0x01,
	0x12, 0x64, 0x0a, 0x17, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x68, 0x65, 0x61, 0x64,
	0x65, 0x72, 0x73, 0x5f, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x2c, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x6d, 0x69, 0x64, 0x64,
	0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2e, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x2e, 0x76,
	0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x52,
	0x15, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x52,
	0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x12, 0x66, 0x0a, 0x18, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x5f, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x5f, 0x72, 0x65, 0x77, 0x72, 0x69,
	0x74, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77,
	0x61, 0x79, 0x2e, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2e, 0x72, 0x65,
	0x77, 0x72, 0x69, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73,
	0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x52, 0x16, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x52, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x12, 0x25,
	0x0a, 0x0b, 0x73, 0x74, 0x72, 0x69, 0x70, 0x50, 0x72, 0x65, 0x66, 0x69, 0x78, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x03, 0x48, 0x01, 0x52, 0x0b, 0x73, 0x74, 0x72, 0x69, 0x70, 0x50, 0x72, 0x65, 0x66,
	0x69, 0x78, 0x88, 0x01, 0x01, 0x42, 0x0f, 0x0a, 0x0d, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x5f, 0x72,
	0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x42, 0x0e, 0x0a, 0x0c, 0x5f, 0x73, 0x74, 0x72, 0x69, 0x70,
	0x50, 0x72, 0x65, 0x66, 0x69, 0x78, 0x42, 0x40, 0x5a, 0x3e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x2d, 0x6b, 0x72, 0x61, 0x74, 0x6f, 0x73, 0x2f, 0x67,
	0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x61, 0x74, 0x65, 0x77,
	0x61, 0x79, 0x2f, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2f, 0x72, 0x65,
	0x77, 0x72, 0x69, 0x74, 0x65, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDescOnce sync.Once
	file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDescData = file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDesc
)

func file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDescGZIP() []byte {
	file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDescOnce.Do(func() {
		file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDescData)
	})
	return file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDescData
}

var file_api_gateway_middleware_rewrite_v1_rewrite_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_api_gateway_middleware_rewrite_v1_rewrite_proto_goTypes = []interface{}{
	(*HeadersPolicy)(nil), // 0: gateway.middleware.rewrite.v1.HeadersPolicy
	(*Rewrite)(nil),       // 1: gateway.middleware.rewrite.v1.Rewrite
	nil,                   // 2: gateway.middleware.rewrite.v1.HeadersPolicy.SetEntry
	nil,                   // 3: gateway.middleware.rewrite.v1.HeadersPolicy.AddEntry
}
var file_api_gateway_middleware_rewrite_v1_rewrite_proto_depIdxs = []int32{
	2, // 0: gateway.middleware.rewrite.v1.HeadersPolicy.set:type_name -> gateway.middleware.rewrite.v1.HeadersPolicy.SetEntry
	3, // 1: gateway.middleware.rewrite.v1.HeadersPolicy.add:type_name -> gateway.middleware.rewrite.v1.HeadersPolicy.AddEntry
	0, // 2: gateway.middleware.rewrite.v1.Rewrite.request_headers_rewrite:type_name -> gateway.middleware.rewrite.v1.HeadersPolicy
	0, // 3: gateway.middleware.rewrite.v1.Rewrite.response_headers_rewrite:type_name -> gateway.middleware.rewrite.v1.HeadersPolicy
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_api_gateway_middleware_rewrite_v1_rewrite_proto_init() }
func file_api_gateway_middleware_rewrite_v1_rewrite_proto_init() {
	if File_api_gateway_middleware_rewrite_v1_rewrite_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_gateway_middleware_rewrite_v1_rewrite_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HeadersPolicy); i {
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
		file_api_gateway_middleware_rewrite_v1_rewrite_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Rewrite); i {
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
	file_api_gateway_middleware_rewrite_v1_rewrite_proto_msgTypes[1].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_api_gateway_middleware_rewrite_v1_rewrite_proto_goTypes,
		DependencyIndexes: file_api_gateway_middleware_rewrite_v1_rewrite_proto_depIdxs,
		MessageInfos:      file_api_gateway_middleware_rewrite_v1_rewrite_proto_msgTypes,
	}.Build()
	File_api_gateway_middleware_rewrite_v1_rewrite_proto = out.File
	file_api_gateway_middleware_rewrite_v1_rewrite_proto_rawDesc = nil
	file_api_gateway_middleware_rewrite_v1_rewrite_proto_goTypes = nil
	file_api_gateway_middleware_rewrite_v1_rewrite_proto_depIdxs = nil
}
