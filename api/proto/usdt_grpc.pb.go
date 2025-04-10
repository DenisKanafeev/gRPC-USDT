//

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.21.12
// source: usdt.proto

package proto

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	RateService_GetRateFromExchange_FullMethodName = "/usdt.RateService/GetRateFromExchange"
)

// RateServiceClient is the client API for RateService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RateServiceClient interface {
	GetRateFromExchange(ctx context.Context, in *GetRateFromExchangeRequest, opts ...grpc.CallOption) (*GetRateFromExchangeResponse, error)
}

type rateServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewRateServiceClient(cc grpc.ClientConnInterface) RateServiceClient {
	return &rateServiceClient{cc}
}

func (c *rateServiceClient) GetRateFromExchange(ctx context.Context, in *GetRateFromExchangeRequest, opts ...grpc.CallOption) (*GetRateFromExchangeResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetRateFromExchangeResponse)
	err := c.cc.Invoke(ctx, RateService_GetRateFromExchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RateServiceServer is the server API for RateService service.
// All implementations must embed UnimplementedRateServiceServer
// for forward compatibility.
type RateServiceServer interface {
	GetRateFromExchange(context.Context, *GetRateFromExchangeRequest) (*GetRateFromExchangeResponse, error)
	mustEmbedUnimplementedRateServiceServer()
}

// UnimplementedRateServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedRateServiceServer struct{}

func (UnimplementedRateServiceServer) GetRateFromExchange(context.Context, *GetRateFromExchangeRequest) (*GetRateFromExchangeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRateFromExchange not implemented")
}
func (UnimplementedRateServiceServer) mustEmbedUnimplementedRateServiceServer() {}
func (UnimplementedRateServiceServer) testEmbeddedByValue()                     {}

// UnsafeRateServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RateServiceServer will
// result in compilation errors.
type UnsafeRateServiceServer interface {
	mustEmbedUnimplementedRateServiceServer()
}

func RegisterRateServiceServer(s grpc.ServiceRegistrar, srv RateServiceServer) {
	// If the following call pancis, it indicates UnimplementedRateServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&RateService_ServiceDesc, srv)
}

func _RateService_GetRateFromExchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRateFromExchangeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RateServiceServer).GetRateFromExchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RateService_GetRateFromExchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RateServiceServer).GetRateFromExchange(ctx, req.(*GetRateFromExchangeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RateService_ServiceDesc is the grpc.ServiceDesc for RateService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RateService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "usdt.RateService",
	HandlerType: (*RateServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetRateFromExchange",
			Handler:    _RateService_GetRateFromExchange_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "usdt.proto",
}
