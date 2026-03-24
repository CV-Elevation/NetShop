package handler

import (
	"context"

	"google.golang.org/grpc"

	cartpb "kuoz/netshop/platform/shared/proto/cart"
	commonpb "kuoz/netshop/platform/shared/proto/common"
	"netshop/services/cart/internal/service"
)

type grpcServer struct {
	cartpb.UnimplementedCartServiceServer
	svc *service.CartService
}

func Register(server *grpc.Server, svc *service.CartService) {
	cartpb.RegisterCartServiceServer(server, &grpcServer{svc: svc})
}

func (s *grpcServer) AddItem(ctx context.Context, req *cartpb.AddItemRequest) (*cartpb.AddItemResponse, error) {
	item, totalItems, err := s.svc.AddItem(ctx, req.GetUserId(), req.GetProductId(), req.GetQuantity())
	if err != nil {
		return nil, err
	}
	return &cartpb.AddItemResponse{
		Item:       item,
		TotalItems: totalItems,
	}, nil
}

func (s *grpcServer) GetCart(ctx context.Context, req *cartpb.GetCartRequest) (*cartpb.GetCartResponse, error) {
	items, totalPrice, totalCount, hasInvalid, err := s.svc.GetCart(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &cartpb.GetCartResponse{
		Items:      items,
		TotalPrice: totalPrice,
		TotalCount: totalCount,
		HasInvalid: hasInvalid,
	}, nil
}

func (s *grpcServer) ClearCart(ctx context.Context, req *cartpb.ClearCartRequest) (*commonpb.Empty, error) {
	return &commonpb.Empty{}, s.svc.ClearCart(ctx, req.GetUserId())
}
