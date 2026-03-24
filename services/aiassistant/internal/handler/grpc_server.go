package handler

import (
	"context"

	aiassistantpb "kuoz/netshop/platform/shared/proto/aiassistant"
	"netshop/services/aiassistant/internal/service"

	"google.golang.org/grpc"
)

type grpcServer struct {
	aiassistantpb.UnimplementedAiAssistantServiceServer
	svc *service.AIAssistantService
}

func Register(server *grpc.Server, svc *service.AIAssistantService) {
	aiassistantpb.RegisterAiAssistantServiceServer(server, &grpcServer{svc: svc})
}

// 用作调试接口
func (s *grpcServer) Chat(ctx context.Context, req *aiassistantpb.ChatRequest) (*aiassistantpb.ChatResponse, error) {
	return s.svc.Chat(ctx, req)
}

// 实际输出常用的接口
func (s *grpcServer) ChatStream(req *aiassistantpb.ChatRequest, stream grpc.ServerStreamingServer[aiassistantpb.ChatChunk]) error {
	return s.svc.ChatStream(stream.Context(), req, func(chunk *aiassistantpb.ChatChunk) error {
		return stream.Send(chunk)
	})
}
