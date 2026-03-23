package handler

import (
	"context"

	"google.golang.org/grpc"

	commonpb "kuoz/netshop/platform/shared/proto/common"
	recommendpb "kuoz/netshop/platform/shared/proto/recommend"
	"netshop/services/recommend/internal/service"
)

type grpcServer struct {
	recommendpb.UnimplementedRecommendServiceServer
	svc *service.RecommendService
}

func Register(server *grpc.Server, svc *service.RecommendService) {
	recommendpb.RegisterRecommendServiceServer(server, &grpcServer{svc: svc})
}

func (s *grpcServer) GetRecommendations(ctx context.Context, req *recommendpb.RecommendRequest) (*recommendpb.RecommendResponse, error) {
	items, strategy, err := s.svc.GetRecommendations(ctx, req)
	if err != nil {
		return nil, err
	}
	return &recommendpb.RecommendResponse{
		Items:    items,
		Strategy: strategy,
	}, nil
}

func (s *grpcServer) RecordBehavior(ctx context.Context, req *recommendpb.BehaviorEvent) (*commonpb.Empty, error) {
	return &commonpb.Empty{}, s.svc.RecordBehavior(ctx, req)
}
