// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: streaming/streaming.proto

package streaming

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
	StreamingService_UploadVideo_FullMethodName        = "/streaming.StreamingService/UploadVideo"
	StreamingService_GetVideo_FullMethodName           = "/streaming.StreamingService/GetVideo"
	StreamingService_Search_FullMethodName             = "/streaming.StreamingService/Search"
	StreamingService_GetRecommendations_FullMethodName = "/streaming.StreamingService/GetRecommendations"
	StreamingService_GetIndexM3U8_FullMethodName       = "/streaming.StreamingService/GetIndexM3U8"
	StreamingService_GetHlsSegment_FullMethodName      = "/streaming.StreamingService/GetHlsSegment"
)

// StreamingServiceClient is the client API for StreamingService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type StreamingServiceClient interface {
	// rpc UploadVideo (stream UploadVideoReq) returns (UploadVideoRes);
	// 这行定义了一个客户端流式 RPC 方法。在 gRPC 中，这意味着客户端可以以流的形式发送多个 UploadVideoReq 消息，而服务端在接收完所有消息后只返回一个 UploadVideoRes 响应。
	// 具体来说：
	//   - Stream 输入：
	//
	// 客户端可以分多次发送消息（比如先发送视频元数据，再连续发送多个视频数据块）。这种方式特别适合传输大文件，因为你不需要一次性将整个文件加载到内存中。
	//   - 单一响应：
	//
	// 服务端在接收完所有消息后，处理完毕，然后只返回一次响应（例如上傳成功与否的信息）。
	// 因此，stream 的用途主要是为了支持大文件或数据分块上传，确保传输过程更高效、稳定，并能减少内存占用。如果你采用这种模式，客户端无需额外操作，只需要按顺序发送消息即可；服务端则依次处理这些消息，直到收到流结束信号，再返回响应。
	UploadVideo(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[UploadVideoReq, UploadVideoRes], error)
	GetVideo(ctx context.Context, in *GetVideoReq, opts ...grpc.CallOption) (*GetVideoRes, error)
	Search(ctx context.Context, in *SearchReq, opts ...grpc.CallOption) (*SearchRes, error)
	GetRecommendations(ctx context.Context, in *GetRecommendationsReq, opts ...grpc.CallOption) (*GetRecommendationsRes, error)
	GetIndexM3U8(ctx context.Context, in *GetIndexM3U8Req, opts ...grpc.CallOption) (*GetIndexM3U8Res, error)
	GetHlsSegment(ctx context.Context, in *GetHlsSegmentReq, opts ...grpc.CallOption) (*GetHlsSegmentRes, error)
}

type streamingServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewStreamingServiceClient(cc grpc.ClientConnInterface) StreamingServiceClient {
	return &streamingServiceClient{cc}
}

func (c *streamingServiceClient) UploadVideo(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[UploadVideoReq, UploadVideoRes], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &StreamingService_ServiceDesc.Streams[0], StreamingService_UploadVideo_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[UploadVideoReq, UploadVideoRes]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type StreamingService_UploadVideoClient = grpc.ClientStreamingClient[UploadVideoReq, UploadVideoRes]

func (c *streamingServiceClient) GetVideo(ctx context.Context, in *GetVideoReq, opts ...grpc.CallOption) (*GetVideoRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetVideoRes)
	err := c.cc.Invoke(ctx, StreamingService_GetVideo_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *streamingServiceClient) Search(ctx context.Context, in *SearchReq, opts ...grpc.CallOption) (*SearchRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SearchRes)
	err := c.cc.Invoke(ctx, StreamingService_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *streamingServiceClient) GetRecommendations(ctx context.Context, in *GetRecommendationsReq, opts ...grpc.CallOption) (*GetRecommendationsRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetRecommendationsRes)
	err := c.cc.Invoke(ctx, StreamingService_GetRecommendations_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *streamingServiceClient) GetIndexM3U8(ctx context.Context, in *GetIndexM3U8Req, opts ...grpc.CallOption) (*GetIndexM3U8Res, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetIndexM3U8Res)
	err := c.cc.Invoke(ctx, StreamingService_GetIndexM3U8_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *streamingServiceClient) GetHlsSegment(ctx context.Context, in *GetHlsSegmentReq, opts ...grpc.CallOption) (*GetHlsSegmentRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetHlsSegmentRes)
	err := c.cc.Invoke(ctx, StreamingService_GetHlsSegment_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// StreamingServiceServer is the server API for StreamingService service.
// All implementations must embed UnimplementedStreamingServiceServer
// for forward compatibility.
type StreamingServiceServer interface {
	// rpc UploadVideo (stream UploadVideoReq) returns (UploadVideoRes);
	// 这行定义了一个客户端流式 RPC 方法。在 gRPC 中，这意味着客户端可以以流的形式发送多个 UploadVideoReq 消息，而服务端在接收完所有消息后只返回一个 UploadVideoRes 响应。
	// 具体来说：
	//   - Stream 输入：
	//
	// 客户端可以分多次发送消息（比如先发送视频元数据，再连续发送多个视频数据块）。这种方式特别适合传输大文件，因为你不需要一次性将整个文件加载到内存中。
	//   - 单一响应：
	//
	// 服务端在接收完所有消息后，处理完毕，然后只返回一次响应（例如上傳成功与否的信息）。
	// 因此，stream 的用途主要是为了支持大文件或数据分块上传，确保传输过程更高效、稳定，并能减少内存占用。如果你采用这种模式，客户端无需额外操作，只需要按顺序发送消息即可；服务端则依次处理这些消息，直到收到流结束信号，再返回响应。
	UploadVideo(grpc.ClientStreamingServer[UploadVideoReq, UploadVideoRes]) error
	GetVideo(context.Context, *GetVideoReq) (*GetVideoRes, error)
	Search(context.Context, *SearchReq) (*SearchRes, error)
	GetRecommendations(context.Context, *GetRecommendationsReq) (*GetRecommendationsRes, error)
	GetIndexM3U8(context.Context, *GetIndexM3U8Req) (*GetIndexM3U8Res, error)
	GetHlsSegment(context.Context, *GetHlsSegmentReq) (*GetHlsSegmentRes, error)
	mustEmbedUnimplementedStreamingServiceServer()
}

// UnimplementedStreamingServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedStreamingServiceServer struct{}

func (UnimplementedStreamingServiceServer) UploadVideo(grpc.ClientStreamingServer[UploadVideoReq, UploadVideoRes]) error {
	return status.Errorf(codes.Unimplemented, "method UploadVideo not implemented")
}
func (UnimplementedStreamingServiceServer) GetVideo(context.Context, *GetVideoReq) (*GetVideoRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVideo not implemented")
}
func (UnimplementedStreamingServiceServer) Search(context.Context, *SearchReq) (*SearchRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedStreamingServiceServer) GetRecommendations(context.Context, *GetRecommendationsReq) (*GetRecommendationsRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRecommendations not implemented")
}
func (UnimplementedStreamingServiceServer) GetIndexM3U8(context.Context, *GetIndexM3U8Req) (*GetIndexM3U8Res, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetIndexM3U8 not implemented")
}
func (UnimplementedStreamingServiceServer) GetHlsSegment(context.Context, *GetHlsSegmentReq) (*GetHlsSegmentRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetHlsSegment not implemented")
}
func (UnimplementedStreamingServiceServer) mustEmbedUnimplementedStreamingServiceServer() {}
func (UnimplementedStreamingServiceServer) testEmbeddedByValue()                          {}

// UnsafeStreamingServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to StreamingServiceServer will
// result in compilation errors.
type UnsafeStreamingServiceServer interface {
	mustEmbedUnimplementedStreamingServiceServer()
}

func RegisterStreamingServiceServer(s grpc.ServiceRegistrar, srv StreamingServiceServer) {
	// If the following call pancis, it indicates UnimplementedStreamingServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&StreamingService_ServiceDesc, srv)
}

func _StreamingService_UploadVideo_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(StreamingServiceServer).UploadVideo(&grpc.GenericServerStream[UploadVideoReq, UploadVideoRes]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type StreamingService_UploadVideoServer = grpc.ClientStreamingServer[UploadVideoReq, UploadVideoRes]

func _StreamingService_GetVideo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetVideoReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StreamingServiceServer).GetVideo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StreamingService_GetVideo_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StreamingServiceServer).GetVideo(ctx, req.(*GetVideoReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _StreamingService_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SearchReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StreamingServiceServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StreamingService_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StreamingServiceServer).Search(ctx, req.(*SearchReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _StreamingService_GetRecommendations_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRecommendationsReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StreamingServiceServer).GetRecommendations(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StreamingService_GetRecommendations_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StreamingServiceServer).GetRecommendations(ctx, req.(*GetRecommendationsReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _StreamingService_GetIndexM3U8_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetIndexM3U8Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StreamingServiceServer).GetIndexM3U8(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StreamingService_GetIndexM3U8_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StreamingServiceServer).GetIndexM3U8(ctx, req.(*GetIndexM3U8Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _StreamingService_GetHlsSegment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetHlsSegmentReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StreamingServiceServer).GetHlsSegment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StreamingService_GetHlsSegment_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StreamingServiceServer).GetHlsSegment(ctx, req.(*GetHlsSegmentReq))
	}
	return interceptor(ctx, in, info, handler)
}

// StreamingService_ServiceDesc is the grpc.ServiceDesc for StreamingService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var StreamingService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "streaming.StreamingService",
	HandlerType: (*StreamingServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetVideo",
			Handler:    _StreamingService_GetVideo_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _StreamingService_Search_Handler,
		},
		{
			MethodName: "GetRecommendations",
			Handler:    _StreamingService_GetRecommendations_Handler,
		},
		{
			MethodName: "GetIndexM3U8",
			Handler:    _StreamingService_GetIndexM3U8_Handler,
		},
		{
			MethodName: "GetHlsSegment",
			Handler:    _StreamingService_GetHlsSegment_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "UploadVideo",
			Handler:       _StreamingService_UploadVideo_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "streaming/streaming.proto",
}
