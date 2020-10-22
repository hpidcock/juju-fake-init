// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package main

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// FakeInitClient is the client API for FakeInit service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FakeInitClient interface {
	Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*StartResponse, error)
	Exec(ctx context.Context, opts ...grpc.CallOption) (FakeInit_ExecClient, error)
	Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error)
	Signal(ctx context.Context, in *SignalRequest, opts ...grpc.CallOption) (*SignalRequest, error)
}

type fakeInitClient struct {
	cc grpc.ClientConnInterface
}

func NewFakeInitClient(cc grpc.ClientConnInterface) FakeInitClient {
	return &fakeInitClient{cc}
}

func (c *fakeInitClient) Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*StartResponse, error) {
	out := new(StartResponse)
	err := c.cc.Invoke(ctx, "/FakeInit/Start", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fakeInitClient) Exec(ctx context.Context, opts ...grpc.CallOption) (FakeInit_ExecClient, error) {
	stream, err := c.cc.NewStream(ctx, &_FakeInit_serviceDesc.Streams[0], "/FakeInit/Exec", opts...)
	if err != nil {
		return nil, err
	}
	x := &fakeInitExecClient{stream}
	return x, nil
}

type FakeInit_ExecClient interface {
	Send(*ExecRequest) error
	Recv() (*ExecResponse, error)
	grpc.ClientStream
}

type fakeInitExecClient struct {
	grpc.ClientStream
}

func (x *fakeInitExecClient) Send(m *ExecRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *fakeInitExecClient) Recv() (*ExecResponse, error) {
	m := new(ExecResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *fakeInitClient) Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error) {
	out := new(PingResponse)
	err := c.cc.Invoke(ctx, "/FakeInit/Ping", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fakeInitClient) Signal(ctx context.Context, in *SignalRequest, opts ...grpc.CallOption) (*SignalRequest, error) {
	out := new(SignalRequest)
	err := c.cc.Invoke(ctx, "/FakeInit/Signal", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FakeInitServer is the server API for FakeInit service.
// All implementations must embed UnimplementedFakeInitServer
// for forward compatibility
type FakeInitServer interface {
	Start(context.Context, *StartRequest) (*StartResponse, error)
	Exec(FakeInit_ExecServer) error
	Ping(context.Context, *PingRequest) (*PingResponse, error)
	Signal(context.Context, *SignalRequest) (*SignalRequest, error)
	mustEmbedUnimplementedFakeInitServer()
}

// UnimplementedFakeInitServer must be embedded to have forward compatible implementations.
type UnimplementedFakeInitServer struct {
}

func (UnimplementedFakeInitServer) Start(context.Context, *StartRequest) (*StartResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Start not implemented")
}
func (UnimplementedFakeInitServer) Exec(FakeInit_ExecServer) error {
	return status.Errorf(codes.Unimplemented, "method Exec not implemented")
}
func (UnimplementedFakeInitServer) Ping(context.Context, *PingRequest) (*PingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedFakeInitServer) Signal(context.Context, *SignalRequest) (*SignalRequest, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Signal not implemented")
}
func (UnimplementedFakeInitServer) mustEmbedUnimplementedFakeInitServer() {}

// UnsafeFakeInitServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to FakeInitServer will
// result in compilation errors.
type UnsafeFakeInitServer interface {
	mustEmbedUnimplementedFakeInitServer()
}

func RegisterFakeInitServer(s grpc.ServiceRegistrar, srv FakeInitServer) {
	s.RegisterService(&_FakeInit_serviceDesc, srv)
}

func _FakeInit_Start_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FakeInitServer).Start(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/FakeInit/Start",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FakeInitServer).Start(ctx, req.(*StartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FakeInit_Exec_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(FakeInitServer).Exec(&fakeInitExecServer{stream})
}

type FakeInit_ExecServer interface {
	Send(*ExecResponse) error
	Recv() (*ExecRequest, error)
	grpc.ServerStream
}

type fakeInitExecServer struct {
	grpc.ServerStream
}

func (x *fakeInitExecServer) Send(m *ExecResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *fakeInitExecServer) Recv() (*ExecRequest, error) {
	m := new(ExecRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _FakeInit_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FakeInitServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/FakeInit/Ping",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FakeInitServer).Ping(ctx, req.(*PingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FakeInit_Signal_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignalRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FakeInitServer).Signal(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/FakeInit/Signal",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FakeInitServer).Signal(ctx, req.(*SignalRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _FakeInit_serviceDesc = grpc.ServiceDesc{
	ServiceName: "FakeInit",
	HandlerType: (*FakeInitServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Start",
			Handler:    _FakeInit_Start_Handler,
		},
		{
			MethodName: "Ping",
			Handler:    _FakeInit_Ping_Handler,
		},
		{
			MethodName: "Signal",
			Handler:    _FakeInit_Signal_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Exec",
			Handler:       _FakeInit_Exec_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "init.proto",
}
