package handler

import (
	"context"

	"google.golang.org/grpc"

	commonpb "kuoz/netshop/platform/shared/proto/common"
	productpb "kuoz/netshop/platform/shared/proto/product"
	"netshop/services/product/internal/service"
)

type grpcServer struct {
	productpb.UnimplementedProductServiceServer
	svc *service.ProductService
}

func Register(server *grpc.Server, svc *service.ProductService) {
	productpb.RegisterProductServiceServer(server, &grpcServer{svc: svc})
}

func (s *grpcServer) GetProduct(ctx context.Context, req *productpb.GetProductRequest) (*commonpb.Product, error) {
	return s.svc.GetProduct(ctx, req.GetId())
}

func (s *grpcServer) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	items, total, err := s.svc.ListProducts(ctx, req.GetCategory(), req.GetPagination().GetPage(), req.GetPagination().GetPageSize())
	if err != nil {
		return nil, err
	}
	return &productpb.ListProductsResponse{
		Items: items,
		Total: total,
	}, nil
}

func (s *grpcServer) SearchProducts(ctx context.Context, req *productpb.SearchProductsRequest) (*productpb.SearchProductsResponse, error) {
	items, total, err := s.svc.SearchProducts(ctx, req)
	if err != nil {
		return nil, err
	}
	return &productpb.SearchProductsResponse{
		Items: items,
		Total: total,
	}, nil
}
