// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: cosmos/mev/v1alpha1/query.proto

package mevv1alpha1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Query_Builder_FullMethodName   = "/cosmos.mev.v1alpha1.Query/Builder"
	Query_Builders_FullMethodName  = "/cosmos.mev.v1alpha1.Query/Builders"
	Query_Proposer_FullMethodName  = "/cosmos.mev.v1alpha1.Query/Proposer"
	Query_Proposers_FullMethodName = "/cosmos.mev.v1alpha1.Query/Proposers"
)

// QueryClient is the client API for Query service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type QueryClient interface {
	// Builder queries a single builder by address.
	Builder(ctx context.Context, in *QueryBuilderRequest, opts ...grpc.CallOption) (*QueryBuilderResponse, error)
	// Builder queries all builders.
	Builders(ctx context.Context, in *QueryBuildersRequest, opts ...grpc.CallOption) (*QueryBuildersResponse, error)
	// Proposer queries a single proposer by address.
	Proposer(ctx context.Context, in *QueryProposerRequest, opts ...grpc.CallOption) (*QueryProposerResponse, error)
	// Proposers queries all proposers.
	Proposers(ctx context.Context, in *QueryProposersRequest, opts ...grpc.CallOption) (*QueryProposersResponse, error)
}

type queryClient struct {
	cc grpc.ClientConnInterface
}

func NewQueryClient(cc grpc.ClientConnInterface) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Builder(ctx context.Context, in *QueryBuilderRequest, opts ...grpc.CallOption) (*QueryBuilderResponse, error) {
	out := new(QueryBuilderResponse)
	err := c.cc.Invoke(ctx, Query_Builder_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Builders(ctx context.Context, in *QueryBuildersRequest, opts ...grpc.CallOption) (*QueryBuildersResponse, error) {
	out := new(QueryBuildersResponse)
	err := c.cc.Invoke(ctx, Query_Builders_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Proposer(ctx context.Context, in *QueryProposerRequest, opts ...grpc.CallOption) (*QueryProposerResponse, error) {
	out := new(QueryProposerResponse)
	err := c.cc.Invoke(ctx, Query_Proposer_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Proposers(ctx context.Context, in *QueryProposersRequest, opts ...grpc.CallOption) (*QueryProposersResponse, error) {
	out := new(QueryProposersResponse)
	err := c.cc.Invoke(ctx, Query_Proposers_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryServer is the server API for Query service.
// All implementations must embed UnimplementedQueryServer
// for forward compatibility
type QueryServer interface {
	// Builder queries a single builder by address.
	Builder(context.Context, *QueryBuilderRequest) (*QueryBuilderResponse, error)
	// Builder queries all builders.
	Builders(context.Context, *QueryBuildersRequest) (*QueryBuildersResponse, error)
	// Proposer queries a single proposer by address.
	Proposer(context.Context, *QueryProposerRequest) (*QueryProposerResponse, error)
	// Proposers queries all proposers.
	Proposers(context.Context, *QueryProposersRequest) (*QueryProposersResponse, error)
	mustEmbedUnimplementedQueryServer()
}

// UnimplementedQueryServer must be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (UnimplementedQueryServer) Builder(context.Context, *QueryBuilderRequest) (*QueryBuilderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Builder not implemented")
}
func (UnimplementedQueryServer) Builders(context.Context, *QueryBuildersRequest) (*QueryBuildersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Builders not implemented")
}
func (UnimplementedQueryServer) Proposer(context.Context, *QueryProposerRequest) (*QueryProposerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Proposer not implemented")
}
func (UnimplementedQueryServer) Proposers(context.Context, *QueryProposersRequest) (*QueryProposersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Proposers not implemented")
}
func (UnimplementedQueryServer) mustEmbedUnimplementedQueryServer() {}

// UnsafeQueryServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to QueryServer will
// result in compilation errors.
type UnsafeQueryServer interface {
	mustEmbedUnimplementedQueryServer()
}

func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&Query_ServiceDesc, srv)
}

func _Query_Builder_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryBuilderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Builder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Builder_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Builder(ctx, req.(*QueryBuilderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Builders_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryBuildersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Builders(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Builders_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Builders(ctx, req.(*QueryBuildersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Proposer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryProposerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Proposer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Proposer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Proposer(ctx, req.(*QueryProposerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Proposers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryProposersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Proposers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Proposers_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Proposers(ctx, req.(*QueryProposersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Query_ServiceDesc is the grpc.ServiceDesc for Query service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Query_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cosmos.mev.v1alpha1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Builder",
			Handler:    _Query_Builder_Handler,
		},
		{
			MethodName: "Builders",
			Handler:    _Query_Builders_Handler,
		},
		{
			MethodName: "Proposer",
			Handler:    _Query_Proposer_Handler,
		},
		{
			MethodName: "Proposers",
			Handler:    _Query_Proposers_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cosmos/mev/v1alpha1/query.proto",
}
