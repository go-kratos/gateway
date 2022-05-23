// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        (unknown)
// source: v1/rewrite.proto

package rewrite

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
		mi := &file_v1_rewrite_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HeadersPolicy) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HeadersPolicy) ProtoMessage() {}

func (x *HeadersPolicy) ProtoReflect() protoreflect.Message {
	mi := &file_v1_rewrite_proto_msgTypes[0]
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
	return file_v1_rewrite_proto_rawDescGZIP(), []int{0}
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

	HostRewrite           *string        `protobuf:"bytes,1,opt,name=host_rewrite,json=hostRewrite,proto3,oneof" json:"host_rewrite,omitempty"`
	PathRewrite           *string        `protobuf:"bytes,2,opt,name=path_rewrite,json=pathRewrite,proto3,oneof" json:"path_rewrite,omitempty"`
	RequestHeadersRewrite *HeadersPolicy `protobuf:"bytes,3,opt,name=request_headers_rewrite,json=requestHeadersRewrite,proto3" json:"request_headers_rewrite,omitempty"`
	ReponseHeadersRewrite *HeadersPolicy `protobuf:"bytes,4,opt,name=reponse_headers_rewrite,json=reponseHeadersRewrite,proto3" json:"reponse_headers_rewrite,omitempty"`
}

func (x *Rewrite) Reset() {
	*x = Rewrite{}
	if protoimpl.UnsafeEnabled {
		mi := &file_v1_rewrite_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Rewrite) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Rewrite) ProtoMessage() {}

func (x *Rewrite) ProtoReflect() protoreflect.Message {
	mi := &file_v1_rewrite_proto_msgTypes[1]
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
	return file_v1_rewrite_proto_rawDescGZIP(), []int{1}
}

func (x *Rewrite) GetHostRewrite() string {
	if x != nil && x.HostRewrite != nil {
		return *x.HostRewrite
	}
	return ""
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

func (x *Rewrite) GetReponseHeadersRewrite() *HeadersPolicy {
	if x != nil {
		return x.ReponseHeadersRewrite
	}
	return nil
}

var File_v1_rewrite_proto protoreflect.FileDescriptor

var file_v1_rewrite_proto_rawDesc = []byte{
	0x0a, 0x10, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x21, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x6d, 0x69, 0x64, 0x64,
	0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2e, 0x72, 0x65, 0x70, 0x6c, 0x61, 0x63, 0x65, 0x70, 0x61,
	0x74, 0x68, 0x2e, 0x76, 0x31, 0x22, 0xb1, 0x02, 0x0a, 0x0d, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72,
	0x73, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x12, 0x4b, 0x0a, 0x03, 0x73, 0x65, 0x74, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x39, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x6d,
	0x69, 0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2e, 0x72, 0x65, 0x70, 0x6c, 0x61, 0x63,
	0x65, 0x70, 0x61, 0x74, 0x68, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73,
	0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x2e, 0x53, 0x65, 0x74, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52,
	0x03, 0x73, 0x65, 0x74, 0x12, 0x4b, 0x0a, 0x03, 0x61, 0x64, 0x64, 0x18, 0x02, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x39, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x6d, 0x69, 0x64, 0x64,
	0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2e, 0x72, 0x65, 0x70, 0x6c, 0x61, 0x63, 0x65, 0x70, 0x61,
	0x74, 0x68, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x50, 0x6f, 0x6c,
	0x69, 0x63, 0x79, 0x2e, 0x41, 0x64, 0x64, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x03, 0x61, 0x64,
	0x64, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x18, 0x03, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x06, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x1a, 0x36, 0x0a, 0x08, 0x53, 0x65, 0x74,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38,
	0x01, 0x1a, 0x36, 0x0a, 0x08, 0x41, 0x64, 0x64, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xcf, 0x02, 0x0a, 0x07, 0x52, 0x65,
	0x77, 0x72, 0x69, 0x74, 0x65, 0x12, 0x26, 0x0a, 0x0c, 0x68, 0x6f, 0x73, 0x74, 0x5f, 0x72, 0x65,
	0x77, 0x72, 0x69, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0b, 0x68,
	0x6f, 0x73, 0x74, 0x52, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x88, 0x01, 0x01, 0x12, 0x26, 0x0a,
	0x0c, 0x70, 0x61, 0x74, 0x68, 0x5f, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x48, 0x01, 0x52, 0x0b, 0x70, 0x61, 0x74, 0x68, 0x52, 0x65, 0x77, 0x72, 0x69,
	0x74, 0x65, 0x88, 0x01, 0x01, 0x12, 0x68, 0x0a, 0x17, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x5f, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x5f, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x30, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79,
	0x2e, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2e, 0x72, 0x65, 0x70, 0x6c,
	0x61, 0x63, 0x65, 0x70, 0x61, 0x74, 0x68, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65,
	0x72, 0x73, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x52, 0x15, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x52, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x12,
	0x68, 0x0a, 0x17, 0x72, 0x65, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x5f, 0x68, 0x65, 0x61, 0x64, 0x65,
	0x72, 0x73, 0x5f, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x30, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x6d, 0x69, 0x64, 0x64, 0x6c,
	0x65, 0x77, 0x61, 0x72, 0x65, 0x2e, 0x72, 0x65, 0x70, 0x6c, 0x61, 0x63, 0x65, 0x70, 0x61, 0x74,
	0x68, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x50, 0x6f, 0x6c, 0x69,
	0x63, 0x79, 0x52, 0x15, 0x72, 0x65, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x48, 0x65, 0x61, 0x64, 0x65,
	0x72, 0x73, 0x52, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x42, 0x0f, 0x0a, 0x0d, 0x5f, 0x68, 0x6f,
	0x73, 0x74, 0x5f, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x42, 0x0f, 0x0a, 0x0d, 0x5f, 0x70,
	0x61, 0x74, 0x68, 0x5f, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x42, 0x41, 0x5a, 0x3f, 0x67,
	0x69, 0x74, 0x2e, 0x62, 0x69, 0x6c, 0x69, 0x62, 0x69, 0x6c, 0x69, 0x2e, 0x63, 0x6f, 0x2f, 0x6d,
	0x69, 0x63, 0x72, 0x6f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2f, 0x67, 0x61, 0x74,
	0x65, 0x77, 0x61, 0x79, 0x2d, 0x77, 0x6f, 0x72, 0x6b, 0x65, 0x72, 0x2f, 0x6d, 0x69, 0x64, 0x64,
	0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2f, 0x72, 0x65, 0x77, 0x72, 0x69, 0x74, 0x65, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_v1_rewrite_proto_rawDescOnce sync.Once
	file_v1_rewrite_proto_rawDescData = file_v1_rewrite_proto_rawDesc
)

func file_v1_rewrite_proto_rawDescGZIP() []byte {
	file_v1_rewrite_proto_rawDescOnce.Do(func() {
		file_v1_rewrite_proto_rawDescData = protoimpl.X.CompressGZIP(file_v1_rewrite_proto_rawDescData)
	})
	return file_v1_rewrite_proto_rawDescData
}

var file_v1_rewrite_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_v1_rewrite_proto_goTypes = []interface{}{
	(*HeadersPolicy)(nil), // 0: gateway.middleware.replacepath.v1.HeadersPolicy
	(*Rewrite)(nil),       // 1: gateway.middleware.replacepath.v1.Rewrite
	nil,                   // 2: gateway.middleware.replacepath.v1.HeadersPolicy.SetEntry
	nil,                   // 3: gateway.middleware.replacepath.v1.HeadersPolicy.AddEntry
}
var file_v1_rewrite_proto_depIdxs = []int32{
	2, // 0: gateway.middleware.replacepath.v1.HeadersPolicy.set:type_name -> gateway.middleware.replacepath.v1.HeadersPolicy.SetEntry
	3, // 1: gateway.middleware.replacepath.v1.HeadersPolicy.add:type_name -> gateway.middleware.replacepath.v1.HeadersPolicy.AddEntry
	0, // 2: gateway.middleware.replacepath.v1.Rewrite.request_headers_rewrite:type_name -> gateway.middleware.replacepath.v1.HeadersPolicy
	0, // 3: gateway.middleware.replacepath.v1.Rewrite.reponse_headers_rewrite:type_name -> gateway.middleware.replacepath.v1.HeadersPolicy
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_v1_rewrite_proto_init() }
func file_v1_rewrite_proto_init() {
	if File_v1_rewrite_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_v1_rewrite_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_v1_rewrite_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
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
	file_v1_rewrite_proto_msgTypes[1].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_v1_rewrite_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_v1_rewrite_proto_goTypes,
		DependencyIndexes: file_v1_rewrite_proto_depIdxs,
		MessageInfos:      file_v1_rewrite_proto_msgTypes,
	}.Build()
	File_v1_rewrite_proto = out.File
	file_v1_rewrite_proto_rawDesc = nil
	file_v1_rewrite_proto_goTypes = nil
	file_v1_rewrite_proto_depIdxs = nil
}
