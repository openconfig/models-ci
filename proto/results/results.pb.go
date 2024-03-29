//
// Copyright 2021 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.21.12
// source: results.proto

// Package results defines a data structure for output results from models-ci.

package results

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

type PyangOutput struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Messages []*PyangMessage `protobuf:"bytes,1,rep,name=messages,proto3" json:"messages,omitempty"`
}

func (x *PyangOutput) Reset() {
	*x = PyangOutput{}
	if protoimpl.UnsafeEnabled {
		mi := &file_results_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PyangOutput) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PyangOutput) ProtoMessage() {}

func (x *PyangOutput) ProtoReflect() protoreflect.Message {
	mi := &file_results_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PyangOutput.ProtoReflect.Descriptor instead.
func (*PyangOutput) Descriptor() ([]byte, []int) {
	return file_results_proto_rawDescGZIP(), []int{0}
}

func (x *PyangOutput) GetMessages() []*PyangMessage {
	if x != nil {
		return x.Messages
	}
	return nil
}

// PyangMessage represents a parsed output from pyang
//
//	This is parsed using the option
//	--msg-template='messages:{{path:"{file}" line:{line} code:"{code}" type:"{type}" level:{level} message:'"'{msg}'}}"
//
// Reference: https://github.com/mbj4668/pyang/blob/master/bin/pyang
type PyangMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Path    string `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	Line    uint32 `protobuf:"varint,2,opt,name=line,proto3" json:"line,omitempty"`
	Code    string `protobuf:"bytes,3,opt,name=code,proto3" json:"code,omitempty"`
	Type    string `protobuf:"bytes,4,opt,name=type,proto3" json:"type,omitempty"`
	Level   uint32 `protobuf:"varint,5,opt,name=level,proto3" json:"level,omitempty"`
	Message string `protobuf:"bytes,6,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *PyangMessage) Reset() {
	*x = PyangMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_results_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PyangMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PyangMessage) ProtoMessage() {}

func (x *PyangMessage) ProtoReflect() protoreflect.Message {
	mi := &file_results_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PyangMessage.ProtoReflect.Descriptor instead.
func (*PyangMessage) Descriptor() ([]byte, []int) {
	return file_results_proto_rawDescGZIP(), []int{1}
}

func (x *PyangMessage) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *PyangMessage) GetLine() uint32 {
	if x != nil {
		return x.Line
	}
	return 0
}

func (x *PyangMessage) GetCode() string {
	if x != nil {
		return x.Code
	}
	return ""
}

func (x *PyangMessage) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *PyangMessage) GetLevel() uint32 {
	if x != nil {
		return x.Level
	}
	return 0
}

func (x *PyangMessage) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

var File_results_proto protoreflect.FileDescriptor

var file_results_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x07, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x22, 0x40, 0x0a, 0x0b, 0x50, 0x79, 0x61, 0x6e,
	0x67, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x12, 0x31, 0x0a, 0x08, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x72, 0x65, 0x73, 0x75,
	0x6c, 0x74, 0x73, 0x2e, 0x50, 0x79, 0x61, 0x6e, 0x67, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x52, 0x08, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x22, 0x8e, 0x01, 0x0a, 0x0c, 0x50,
	0x79, 0x61, 0x6e, 0x67, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x70,
	0x61, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68, 0x12,
	0x12, 0x0a, 0x04, 0x6c, 0x69, 0x6e, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04, 0x6c,
	0x69, 0x6e, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x6c,
	0x65, 0x76, 0x65, 0x6c, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x6c, 0x65, 0x76, 0x65,
	0x6c, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x42, 0x2f, 0x5a, 0x2d, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2d, 0x63, 0x69, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_results_proto_rawDescOnce sync.Once
	file_results_proto_rawDescData = file_results_proto_rawDesc
)

func file_results_proto_rawDescGZIP() []byte {
	file_results_proto_rawDescOnce.Do(func() {
		file_results_proto_rawDescData = protoimpl.X.CompressGZIP(file_results_proto_rawDescData)
	})
	return file_results_proto_rawDescData
}

var file_results_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_results_proto_goTypes = []interface{}{
	(*PyangOutput)(nil),  // 0: results.PyangOutput
	(*PyangMessage)(nil), // 1: results.PyangMessage
}
var file_results_proto_depIdxs = []int32{
	1, // 0: results.PyangOutput.messages:type_name -> results.PyangMessage
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_results_proto_init() }
func file_results_proto_init() {
	if File_results_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_results_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PyangOutput); i {
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
		file_results_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PyangMessage); i {
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
			RawDescriptor: file_results_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_results_proto_goTypes,
		DependencyIndexes: file_results_proto_depIdxs,
		MessageInfos:      file_results_proto_msgTypes,
	}.Build()
	File_results_proto = out.File
	file_results_proto_rawDesc = nil
	file_results_proto_goTypes = nil
	file_results_proto_depIdxs = nil
}
