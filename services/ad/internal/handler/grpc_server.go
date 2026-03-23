package handler

import (
	"context"

	"google.golang.org/grpc"

	adpb "kuoz/netshop/platform/shared/proto/ad"
	commonpb "kuoz/netshop/platform/shared/proto/common"
	"netshop/services/ad/internal/service"
)

type grpcServer struct {
	adpb.UnimplementedAdServiceServer
	svc *service.AdService
}

func Register(server *grpc.Server, svc *service.AdService) {
	adpb.RegisterAdServiceServer(server, &grpcServer{svc: svc})
}

func (s *grpcServer) GetAds(ctx context.Context, req *adpb.GetAdsRequest) (*adpb.GetAdsResponse, error) {
	items, err := s.svc.GetAds(ctx, req)
	if err != nil {
		return nil, err
	}
	return &adpb.GetAdsResponse{Items: items}, nil
}

func (s *grpcServer) RecordAdEvent(ctx context.Context, req *adpb.AdEvent) (*commonpb.Empty, error) {
	return &commonpb.Empty{}, s.svc.RecordAdEvent(ctx, req)
}
