// Code generated by protoc-gen-go. DO NOT EDIT.
// source: src/proto/grpc/core/stats.proto

package core

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Bucket struct {
	Start                float64  `protobuf:"fixed64,1,opt,name=start,proto3" json:"start,omitempty"`
	Count                uint64   `protobuf:"varint,2,opt,name=count,proto3" json:"count,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Bucket) Reset()         { *m = Bucket{} }
func (m *Bucket) String() string { return proto.CompactTextString(m) }
func (*Bucket) ProtoMessage()    {}
func (*Bucket) Descriptor() ([]byte, []int) {
	return fileDescriptor_fae6f913c43aaa6e, []int{0}
}

func (m *Bucket) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Bucket.Unmarshal(m, b)
}
func (m *Bucket) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Bucket.Marshal(b, m, deterministic)
}
func (m *Bucket) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Bucket.Merge(m, src)
}
func (m *Bucket) XXX_Size() int {
	return xxx_messageInfo_Bucket.Size(m)
}
func (m *Bucket) XXX_DiscardUnknown() {
	xxx_messageInfo_Bucket.DiscardUnknown(m)
}

var xxx_messageInfo_Bucket proto.InternalMessageInfo

func (m *Bucket) GetStart() float64 {
	if m != nil {
		return m.Start
	}
	return 0
}

func (m *Bucket) GetCount() uint64 {
	if m != nil {
		return m.Count
	}
	return 0
}

type Histogram struct {
	Buckets              []*Bucket `protobuf:"bytes,1,rep,name=buckets,proto3" json:"buckets,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *Histogram) Reset()         { *m = Histogram{} }
func (m *Histogram) String() string { return proto.CompactTextString(m) }
func (*Histogram) ProtoMessage()    {}
func (*Histogram) Descriptor() ([]byte, []int) {
	return fileDescriptor_fae6f913c43aaa6e, []int{1}
}

func (m *Histogram) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Histogram.Unmarshal(m, b)
}
func (m *Histogram) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Histogram.Marshal(b, m, deterministic)
}
func (m *Histogram) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Histogram.Merge(m, src)
}
func (m *Histogram) XXX_Size() int {
	return xxx_messageInfo_Histogram.Size(m)
}
func (m *Histogram) XXX_DiscardUnknown() {
	xxx_messageInfo_Histogram.DiscardUnknown(m)
}

var xxx_messageInfo_Histogram proto.InternalMessageInfo

func (m *Histogram) GetBuckets() []*Bucket {
	if m != nil {
		return m.Buckets
	}
	return nil
}

type Metric struct {
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Types that are valid to be assigned to Value:
	//	*Metric_Count
	//	*Metric_Histogram
	Value                isMetric_Value `protobuf_oneof:"value"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *Metric) Reset()         { *m = Metric{} }
func (m *Metric) String() string { return proto.CompactTextString(m) }
func (*Metric) ProtoMessage()    {}
func (*Metric) Descriptor() ([]byte, []int) {
	return fileDescriptor_fae6f913c43aaa6e, []int{2}
}

func (m *Metric) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Metric.Unmarshal(m, b)
}
func (m *Metric) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Metric.Marshal(b, m, deterministic)
}
func (m *Metric) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Metric.Merge(m, src)
}
func (m *Metric) XXX_Size() int {
	return xxx_messageInfo_Metric.Size(m)
}
func (m *Metric) XXX_DiscardUnknown() {
	xxx_messageInfo_Metric.DiscardUnknown(m)
}

var xxx_messageInfo_Metric proto.InternalMessageInfo

func (m *Metric) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

type isMetric_Value interface {
	isMetric_Value()
}

type Metric_Count struct {
	Count uint64 `protobuf:"varint,10,opt,name=count,proto3,oneof"`
}

type Metric_Histogram struct {
	Histogram *Histogram `protobuf:"bytes,11,opt,name=histogram,proto3,oneof"`
}

func (*Metric_Count) isMetric_Value() {}

func (*Metric_Histogram) isMetric_Value() {}

func (m *Metric) GetValue() isMetric_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *Metric) GetCount() uint64 {
	if x, ok := m.GetValue().(*Metric_Count); ok {
		return x.Count
	}
	return 0
}

func (m *Metric) GetHistogram() *Histogram {
	if x, ok := m.GetValue().(*Metric_Histogram); ok {
		return x.Histogram
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*Metric) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*Metric_Count)(nil),
		(*Metric_Histogram)(nil),
	}
}

type Stats struct {
	Metrics              []*Metric `protobuf:"bytes,1,rep,name=metrics,proto3" json:"metrics,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *Stats) Reset()         { *m = Stats{} }
func (m *Stats) String() string { return proto.CompactTextString(m) }
func (*Stats) ProtoMessage()    {}
func (*Stats) Descriptor() ([]byte, []int) {
	return fileDescriptor_fae6f913c43aaa6e, []int{3}
}

func (m *Stats) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Stats.Unmarshal(m, b)
}
func (m *Stats) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Stats.Marshal(b, m, deterministic)
}
func (m *Stats) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Stats.Merge(m, src)
}
func (m *Stats) XXX_Size() int {
	return xxx_messageInfo_Stats.Size(m)
}
func (m *Stats) XXX_DiscardUnknown() {
	xxx_messageInfo_Stats.DiscardUnknown(m)
}

var xxx_messageInfo_Stats proto.InternalMessageInfo

func (m *Stats) GetMetrics() []*Metric {
	if m != nil {
		return m.Metrics
	}
	return nil
}

func init() {
	proto.RegisterType((*Bucket)(nil), "grpc.core.Bucket")
	proto.RegisterType((*Histogram)(nil), "grpc.core.Histogram")
	proto.RegisterType((*Metric)(nil), "grpc.core.Metric")
	proto.RegisterType((*Stats)(nil), "grpc.core.Stats")
}

func init() { proto.RegisterFile("src/proto/grpc/core/stats.proto", fileDescriptor_fae6f913c43aaa6e) }

var fileDescriptor_fae6f913c43aaa6e = []byte{
	// 253 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x90, 0xb1, 0x4b, 0xc4, 0x30,
	0x14, 0xc6, 0x2f, 0x7a, 0xed, 0xd1, 0xd7, 0xc9, 0x70, 0x48, 0x37, 0x4b, 0xa7, 0x82, 0x90, 0x48,
	0xbd, 0xc1, 0xb9, 0x53, 0x17, 0x97, 0xb8, 0xb9, 0xa5, 0x21, 0xf4, 0x8a, 0xd7, 0xe6, 0x48, 0x5e,
	0xfd, 0xfb, 0x25, 0x89, 0xad, 0x22, 0x2e, 0x21, 0xef, 0xe3, 0xe3, 0xfb, 0x7d, 0xef, 0xc1, 0x83,
	0xb3, 0x8a, 0x5f, 0xad, 0x41, 0xc3, 0x07, 0x7b, 0x55, 0x5c, 0x19, 0xab, 0xb9, 0x43, 0x89, 0x8e,
	0x05, 0x95, 0x66, 0x5e, 0x66, 0x5e, 0xae, 0x4e, 0x90, 0xb6, 0x8b, 0xfa, 0xd0, 0x48, 0x8f, 0x90,
	0x38, 0x94, 0x16, 0x0b, 0x52, 0x92, 0x9a, 0x88, 0x38, 0x78, 0x55, 0x99, 0x65, 0xc6, 0xe2, 0xa6,
	0x24, 0xf5, 0x5e, 0xc4, 0xa1, 0x7a, 0x81, 0xac, 0x1b, 0x1d, 0x9a, 0xc1, 0xca, 0x89, 0x3e, 0xc2,
	0xa1, 0x0f, 0x11, 0xae, 0x20, 0xe5, 0x6d, 0x9d, 0x37, 0x77, 0x6c, 0xcb, 0x67, 0x31, 0x5c, 0xac,
	0x8e, 0xca, 0x41, 0xfa, 0xaa, 0xd1, 0x8e, 0x8a, 0x52, 0xd8, 0xcf, 0x72, 0xd2, 0x01, 0x97, 0x89,
	0xf0, 0xa7, 0xf7, 0x2b, 0x0d, 0x3c, 0xad, 0xdb, 0x7d, 0xf3, 0xe8, 0x09, 0xb2, 0xf3, 0xca, 0x2b,
	0xf2, 0x92, 0xd4, 0x79, 0x73, 0xfc, 0x05, 0xd9, 0xba, 0x74, 0x3b, 0xf1, 0x63, 0x6c, 0x0f, 0x90,
	0x7c, 0xca, 0xcb, 0xe2, 0x97, 0x4c, 0xde, 0xfc, 0xfa, 0xbe, 0xea, 0x14, 0xe8, 0xff, 0x55, 0x8d,
	0xbd, 0xc4, 0xea, 0x68, 0x9b, 0xf7, 0xa7, 0x61, 0xc4, 0xf3, 0xd2, 0x33, 0x65, 0xa6, 0x78, 0xc9,
	0xf0, 0xa0, 0x76, 0xa8, 0xd0, 0x5e, 0xf8, 0xa0, 0xe7, 0x3f, 0x57, 0xee, 0xd3, 0x20, 0x3c, 0x7f,
	0x05, 0x00, 0x00, 0xff, 0xff, 0xa3, 0x4e, 0xbe, 0x94, 0x83, 0x01, 0x00, 0x00,
}
