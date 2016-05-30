// Code generated by protoc-gen-go.
// source: groupcache.proto
// DO NOT EDIT!

/*
Package main is a generated protocol buffer package.

It is generated from these files:
	groupcache.proto

It has these top-level messages:
	BuildJekyllResponse
	BuiltFileResponse
*/
package main

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type BuildJekyllResponse struct {
	Error string `protobuf:"bytes,1,opt,name=error" json:"error,omitempty"`
	Code  int32  `protobuf:"varint,2,opt,name=code" json:"code,omitempty"`
}

func (m *BuildJekyllResponse) Reset()                    { *m = BuildJekyllResponse{} }
func (m *BuildJekyllResponse) String() string            { return proto.CompactTextString(m) }
func (*BuildJekyllResponse) ProtoMessage()               {}
func (*BuildJekyllResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type BuiltFileResponse struct {
	Data    []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	ModTime int64  `protobuf:"varint,2,opt,name=modTime" json:"modTime,omitempty"`
	Error   string `protobuf:"bytes,3,opt,name=error" json:"error,omitempty"`
	Code    int32  `protobuf:"varint,4,opt,name=code" json:"code,omitempty"`
}

func (m *BuiltFileResponse) Reset()                    { *m = BuiltFileResponse{} }
func (m *BuiltFileResponse) String() string            { return proto.CompactTextString(m) }
func (*BuiltFileResponse) ProtoMessage()               {}
func (*BuiltFileResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func init() {
	proto.RegisterType((*BuildJekyllResponse)(nil), "main.BuildJekyllResponse")
	proto.RegisterType((*BuiltFileResponse)(nil), "main.BuiltFileResponse")
}

var fileDescriptor0 = []byte{
	// 159 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0x12, 0x48, 0x2f, 0xca, 0x2f,
	0x2d, 0x48, 0x4e, 0x4c, 0xce, 0x48, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0xc9, 0x4d,
	0xcc, 0xcc, 0x53, 0xb2, 0xe7, 0x12, 0x76, 0x2a, 0xcd, 0xcc, 0x49, 0xf1, 0x4a, 0xcd, 0xae, 0xcc,
	0xc9, 0x09, 0x4a, 0x2d, 0x2e, 0xc8, 0xcf, 0x2b, 0x4e, 0x15, 0x12, 0xe1, 0x62, 0x4d, 0x2d, 0x2a,
	0xca, 0x2f, 0x92, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x0c, 0x82, 0x70, 0x84, 0x84, 0xb8, 0x58, 0x92,
	0xf3, 0x53, 0x52, 0x25, 0x98, 0x80, 0x82, 0xac, 0x41, 0x60, 0xb6, 0x52, 0x36, 0x97, 0x20, 0xc8,
	0x80, 0x12, 0xb7, 0xcc, 0x9c, 0x54, 0xb8, 0x76, 0xa0, 0xc2, 0x94, 0xc4, 0x92, 0x44, 0xb0, 0x6e,
	0x9e, 0x20, 0x30, 0x5b, 0x48, 0x82, 0x8b, 0x3d, 0x37, 0x3f, 0x25, 0x24, 0x33, 0x17, 0xa2, 0x9f,
	0x39, 0x08, 0xc6, 0x45, 0x58, 0xc6, 0x8c, 0xcd, 0x32, 0x16, 0x84, 0x65, 0x49, 0x6c, 0x60, 0xa7,
	0x1b, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0xcf, 0xed, 0x5b, 0x1f, 0xce, 0x00, 0x00, 0x00,
}
