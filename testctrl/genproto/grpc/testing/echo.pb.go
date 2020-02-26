// Code generated by protoc-gen-go. DO NOT EDIT.
// source: src/proto/grpc/testing/echo.proto

package testing

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

func init() { proto.RegisterFile("src/proto/grpc/testing/echo.proto", fileDescriptor_0bb4a9b5ff27ef2c) }

var fileDescriptor_0bb4a9b5ff27ef2c = []byte{
	// 291 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x92, 0xbf, 0x4f, 0xfb, 0x30,
	0x10, 0xc5, 0x15, 0x7d, 0xbf, 0x62, 0x38, 0xb5, 0x45, 0xf2, 0x04, 0x81, 0x85, 0x0d, 0x21, 0x94,
	0x54, 0xad, 0x18, 0x59, 0x5a, 0xf1, 0xa3, 0x03, 0x0c, 0x0d, 0x08, 0x89, 0x05, 0x39, 0xce, 0x29,
	0xb1, 0x48, 0x6c, 0xe3, 0xbb, 0xf2, 0x6f, 0xf2, 0x2f, 0xa1, 0xa4, 0x09, 0xa2, 0x50, 0x18, 0x5a,
	0x46, 0xbf, 0x77, 0xf7, 0xf1, 0xb3, 0xfc, 0xe0, 0x88, 0xbc, 0x8a, 0x9d, 0xb7, 0x6c, 0xe3, 0xdc,
	0x3b, 0x15, 0x33, 0x12, 0x6b, 0x93, 0xc7, 0xa8, 0x0a, 0x1b, 0x35, 0xba, 0xe8, 0xd5, 0x46, 0xd4,
	0x1a, 0xe1, 0xc9, 0x2f, 0x0b, 0x4f, 0x15, 0x12, 0xc9, 0x1c, 0x69, 0xb9, 0x19, 0x9e, 0xfe, 0x30,
	0x4b, 0xba, 0x72, 0x25, 0x7e, 0x99, 0x1e, 0xbd, 0xfd, 0x83, 0xdd, 0x0b, 0x55, 0xd8, 0x3b, 0x24,
	0x4e, 0xd0, 0xbf, 0x6a, 0x85, 0xe2, 0x1c, 0xfe, 0xd7, 0x92, 0xd8, 0x8f, 0x3e, 0x87, 0x88, 0x6a,
	0x6d, 0x8e, 0x2f, 0x0b, 0x24, 0x0e, 0xc3, 0x75, 0x16, 0x39, 0x6b, 0x08, 0xc5, 0x03, 0x84, 0xd3,
	0x02, 0xd5, 0xf3, 0xb4, 0xd4, 0x68, 0x78, 0x66, 0x34, 0x6b, 0x59, 0xde, 0x20, 0xcb, 0x4c, 0xb2,
	0x14, 0x07, 0xab, 0x9b, 0x49, 0x93, 0xaa, 0xc3, 0x1e, 0xae, 0x37, 0x5b, 0xf0, 0x35, 0xf4, 0xdb,
	0xc1, 0x84, 0x3d, 0xca, 0x6a, 0xc3, 0x80, 0xc7, 0x81, 0x98, 0xc1, 0xa0, 0x3b, 0x6d, 0x85, 0x1a,
	0x06, 0xe2, 0x0a, 0x60, 0xa2, 0x33, 0xbd, 0x65, 0xa2, 0x61, 0x20, 0x2e, 0xa1, 0x7f, 0x6f, 0x9a,
	0x07, 0x57, 0x68, 0x18, 0xb3, 0x0d, 0x59, 0xa3, 0x14, 0xf6, 0x56, 0x38, 0xb5, 0xd9, 0xfd, 0xec,
	0x5f, 0xdd, 0x31, 0x80, 0xde, 0xad, 0x9d, 0x3b, 0xd5, 0x72, 0x27, 0x67, 0x8f, 0xe3, 0x5c, 0x73,
	0xb1, 0x48, 0x23, 0x65, 0xab, 0x65, 0xed, 0x3e, 0xba, 0xa7, 0xd8, 0x97, 0x71, 0x8e, 0xe6, 0x7b,
	0x25, 0xd3, 0x9d, 0x46, 0x1b, 0xbf, 0x07, 0x00, 0x00, 0xff, 0xff, 0x05, 0x8d, 0x5b, 0x5f, 0x10,
	0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// EchoTestServiceClient is the client API for EchoTestService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type EchoTestServiceClient interface {
	Echo(ctx context.Context, in *EchoRequest, opts ...grpc.CallOption) (*EchoResponse, error)
	// A service which checks that the initial metadata sent over contains some
	// expected key value pair
	CheckClientInitialMetadata(ctx context.Context, in *SimpleRequest, opts ...grpc.CallOption) (*SimpleResponse, error)
	RequestStream(ctx context.Context, opts ...grpc.CallOption) (EchoTestService_RequestStreamClient, error)
	ResponseStream(ctx context.Context, in *EchoRequest, opts ...grpc.CallOption) (EchoTestService_ResponseStreamClient, error)
	BidiStream(ctx context.Context, opts ...grpc.CallOption) (EchoTestService_BidiStreamClient, error)
	Unimplemented(ctx context.Context, in *EchoRequest, opts ...grpc.CallOption) (*EchoResponse, error)
}

type echoTestServiceClient struct {
	cc *grpc.ClientConn
}

func NewEchoTestServiceClient(cc *grpc.ClientConn) EchoTestServiceClient {
	return &echoTestServiceClient{cc}
}

func (c *echoTestServiceClient) Echo(ctx context.Context, in *EchoRequest, opts ...grpc.CallOption) (*EchoResponse, error) {
	out := new(EchoResponse)
	err := c.cc.Invoke(ctx, "/grpc.testing.EchoTestService/Echo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *echoTestServiceClient) CheckClientInitialMetadata(ctx context.Context, in *SimpleRequest, opts ...grpc.CallOption) (*SimpleResponse, error) {
	out := new(SimpleResponse)
	err := c.cc.Invoke(ctx, "/grpc.testing.EchoTestService/CheckClientInitialMetadata", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *echoTestServiceClient) RequestStream(ctx context.Context, opts ...grpc.CallOption) (EchoTestService_RequestStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &_EchoTestService_serviceDesc.Streams[0], "/grpc.testing.EchoTestService/RequestStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &echoTestServiceRequestStreamClient{stream}
	return x, nil
}

type EchoTestService_RequestStreamClient interface {
	Send(*EchoRequest) error
	CloseAndRecv() (*EchoResponse, error)
	grpc.ClientStream
}

type echoTestServiceRequestStreamClient struct {
	grpc.ClientStream
}

func (x *echoTestServiceRequestStreamClient) Send(m *EchoRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *echoTestServiceRequestStreamClient) CloseAndRecv() (*EchoResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(EchoResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *echoTestServiceClient) ResponseStream(ctx context.Context, in *EchoRequest, opts ...grpc.CallOption) (EchoTestService_ResponseStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &_EchoTestService_serviceDesc.Streams[1], "/grpc.testing.EchoTestService/ResponseStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &echoTestServiceResponseStreamClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type EchoTestService_ResponseStreamClient interface {
	Recv() (*EchoResponse, error)
	grpc.ClientStream
}

type echoTestServiceResponseStreamClient struct {
	grpc.ClientStream
}

func (x *echoTestServiceResponseStreamClient) Recv() (*EchoResponse, error) {
	m := new(EchoResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *echoTestServiceClient) BidiStream(ctx context.Context, opts ...grpc.CallOption) (EchoTestService_BidiStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &_EchoTestService_serviceDesc.Streams[2], "/grpc.testing.EchoTestService/BidiStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &echoTestServiceBidiStreamClient{stream}
	return x, nil
}

type EchoTestService_BidiStreamClient interface {
	Send(*EchoRequest) error
	Recv() (*EchoResponse, error)
	grpc.ClientStream
}

type echoTestServiceBidiStreamClient struct {
	grpc.ClientStream
}

func (x *echoTestServiceBidiStreamClient) Send(m *EchoRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *echoTestServiceBidiStreamClient) Recv() (*EchoResponse, error) {
	m := new(EchoResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *echoTestServiceClient) Unimplemented(ctx context.Context, in *EchoRequest, opts ...grpc.CallOption) (*EchoResponse, error) {
	out := new(EchoResponse)
	err := c.cc.Invoke(ctx, "/grpc.testing.EchoTestService/Unimplemented", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EchoTestServiceServer is the server API for EchoTestService service.
type EchoTestServiceServer interface {
	Echo(context.Context, *EchoRequest) (*EchoResponse, error)
	// A service which checks that the initial metadata sent over contains some
	// expected key value pair
	CheckClientInitialMetadata(context.Context, *SimpleRequest) (*SimpleResponse, error)
	RequestStream(EchoTestService_RequestStreamServer) error
	ResponseStream(*EchoRequest, EchoTestService_ResponseStreamServer) error
	BidiStream(EchoTestService_BidiStreamServer) error
	Unimplemented(context.Context, *EchoRequest) (*EchoResponse, error)
}

// UnimplementedEchoTestServiceServer can be embedded to have forward compatible implementations.
type UnimplementedEchoTestServiceServer struct {
}

func (*UnimplementedEchoTestServiceServer) Echo(ctx context.Context, req *EchoRequest) (*EchoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Echo not implemented")
}
func (*UnimplementedEchoTestServiceServer) CheckClientInitialMetadata(ctx context.Context, req *SimpleRequest) (*SimpleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckClientInitialMetadata not implemented")
}
func (*UnimplementedEchoTestServiceServer) RequestStream(srv EchoTestService_RequestStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method RequestStream not implemented")
}
func (*UnimplementedEchoTestServiceServer) ResponseStream(req *EchoRequest, srv EchoTestService_ResponseStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method ResponseStream not implemented")
}
func (*UnimplementedEchoTestServiceServer) BidiStream(srv EchoTestService_BidiStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method BidiStream not implemented")
}
func (*UnimplementedEchoTestServiceServer) Unimplemented(ctx context.Context, req *EchoRequest) (*EchoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Unimplemented not implemented")
}

func RegisterEchoTestServiceServer(s *grpc.Server, srv EchoTestServiceServer) {
	s.RegisterService(&_EchoTestService_serviceDesc, srv)
}

func _EchoTestService_Echo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EchoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EchoTestServiceServer).Echo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/grpc.testing.EchoTestService/Echo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EchoTestServiceServer).Echo(ctx, req.(*EchoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EchoTestService_CheckClientInitialMetadata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SimpleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EchoTestServiceServer).CheckClientInitialMetadata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/grpc.testing.EchoTestService/CheckClientInitialMetadata",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EchoTestServiceServer).CheckClientInitialMetadata(ctx, req.(*SimpleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EchoTestService_RequestStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EchoTestServiceServer).RequestStream(&echoTestServiceRequestStreamServer{stream})
}

type EchoTestService_RequestStreamServer interface {
	SendAndClose(*EchoResponse) error
	Recv() (*EchoRequest, error)
	grpc.ServerStream
}

type echoTestServiceRequestStreamServer struct {
	grpc.ServerStream
}

func (x *echoTestServiceRequestStreamServer) SendAndClose(m *EchoResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *echoTestServiceRequestStreamServer) Recv() (*EchoRequest, error) {
	m := new(EchoRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _EchoTestService_ResponseStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(EchoRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(EchoTestServiceServer).ResponseStream(m, &echoTestServiceResponseStreamServer{stream})
}

type EchoTestService_ResponseStreamServer interface {
	Send(*EchoResponse) error
	grpc.ServerStream
}

type echoTestServiceResponseStreamServer struct {
	grpc.ServerStream
}

func (x *echoTestServiceResponseStreamServer) Send(m *EchoResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _EchoTestService_BidiStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EchoTestServiceServer).BidiStream(&echoTestServiceBidiStreamServer{stream})
}

type EchoTestService_BidiStreamServer interface {
	Send(*EchoResponse) error
	Recv() (*EchoRequest, error)
	grpc.ServerStream
}

type echoTestServiceBidiStreamServer struct {
	grpc.ServerStream
}

func (x *echoTestServiceBidiStreamServer) Send(m *EchoResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *echoTestServiceBidiStreamServer) Recv() (*EchoRequest, error) {
	m := new(EchoRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _EchoTestService_Unimplemented_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EchoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EchoTestServiceServer).Unimplemented(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/grpc.testing.EchoTestService/Unimplemented",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EchoTestServiceServer).Unimplemented(ctx, req.(*EchoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _EchoTestService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "grpc.testing.EchoTestService",
	HandlerType: (*EchoTestServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Echo",
			Handler:    _EchoTestService_Echo_Handler,
		},
		{
			MethodName: "CheckClientInitialMetadata",
			Handler:    _EchoTestService_CheckClientInitialMetadata_Handler,
		},
		{
			MethodName: "Unimplemented",
			Handler:    _EchoTestService_Unimplemented_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "RequestStream",
			Handler:       _EchoTestService_RequestStream_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "ResponseStream",
			Handler:       _EchoTestService_ResponseStream_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "BidiStream",
			Handler:       _EchoTestService_BidiStream_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "src/proto/grpc/testing/echo.proto",
}

// UnimplementedEchoServiceClient is the client API for UnimplementedEchoService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type UnimplementedEchoServiceClient interface {
	Unimplemented(ctx context.Context, in *EchoRequest, opts ...grpc.CallOption) (*EchoResponse, error)
}

type unimplementedEchoServiceClient struct {
	cc *grpc.ClientConn
}

func NewUnimplementedEchoServiceClient(cc *grpc.ClientConn) UnimplementedEchoServiceClient {
	return &unimplementedEchoServiceClient{cc}
}

func (c *unimplementedEchoServiceClient) Unimplemented(ctx context.Context, in *EchoRequest, opts ...grpc.CallOption) (*EchoResponse, error) {
	out := new(EchoResponse)
	err := c.cc.Invoke(ctx, "/grpc.testing.UnimplementedEchoService/Unimplemented", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UnimplementedEchoServiceServer is the server API for UnimplementedEchoService service.
type UnimplementedEchoServiceServer interface {
	Unimplemented(context.Context, *EchoRequest) (*EchoResponse, error)
}

// UnimplementedUnimplementedEchoServiceServer can be embedded to have forward compatible implementations.
type UnimplementedUnimplementedEchoServiceServer struct {
}

func (*UnimplementedUnimplementedEchoServiceServer) Unimplemented(ctx context.Context, req *EchoRequest) (*EchoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Unimplemented not implemented")
}

func RegisterUnimplementedEchoServiceServer(s *grpc.Server, srv UnimplementedEchoServiceServer) {
	s.RegisterService(&_UnimplementedEchoService_serviceDesc, srv)
}

func _UnimplementedEchoService_Unimplemented_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EchoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UnimplementedEchoServiceServer).Unimplemented(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/grpc.testing.UnimplementedEchoService/Unimplemented",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UnimplementedEchoServiceServer).Unimplemented(ctx, req.(*EchoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _UnimplementedEchoService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "grpc.testing.UnimplementedEchoService",
	HandlerType: (*UnimplementedEchoServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Unimplemented",
			Handler:    _UnimplementedEchoService_Unimplemented_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "src/proto/grpc/testing/echo.proto",
}

// NoRpcServiceClient is the client API for NoRpcService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type NoRpcServiceClient interface {
}

type noRpcServiceClient struct {
	cc *grpc.ClientConn
}

func NewNoRpcServiceClient(cc *grpc.ClientConn) NoRpcServiceClient {
	return &noRpcServiceClient{cc}
}

// NoRpcServiceServer is the server API for NoRpcService service.
type NoRpcServiceServer interface {
}

// UnimplementedNoRpcServiceServer can be embedded to have forward compatible implementations.
type UnimplementedNoRpcServiceServer struct {
}

func RegisterNoRpcServiceServer(s *grpc.Server, srv NoRpcServiceServer) {
	s.RegisterService(&_NoRpcService_serviceDesc, srv)
}

var _NoRpcService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "grpc.testing.NoRpcService",
	HandlerType: (*NoRpcServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams:     []grpc.StreamDesc{},
	Metadata:    "src/proto/grpc/testing/echo.proto",
}
