package service

import (
	"context"
	"log"

	"netshop/services/product/internal/repository"

	commonpb "kuoz/netshop/platform/shared/proto/common"
	productpb "kuoz/netshop/platform/shared/proto/product"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProductService struct {
	repo repository.Repository
}

func NewProductService(repo repository.Repository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) GetProduct(ctx context.Context, id string) (*commonpb.Product, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	p, ok, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Printf("GetByID error: %v", err)
		return nil, status.Error(codes.Internal, "db error")
	}
	if !ok {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	return toProto(p), nil
}

func (s *ProductService) ListProducts(ctx context.Context, category string, page, pageSize int32) ([]*commonpb.Product, int32, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	products, total, err := s.repo.List(ctx, repository.ListFilter{
		Category: category,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, 0, status.Error(codes.Internal, "db error")
	}
	return toProtoList(products), total, nil
}

func (s *ProductService) SearchProducts(ctx context.Context, req *productpb.SearchProductsRequest) ([]*commonpb.Product, int32, error) {
	page := req.Pagination.GetPage()
	pageSize := req.Pagination.GetPageSize()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	products, total, err := s.repo.Search(ctx, repository.SearchFilter{
		Keyword:  req.Keyword,
		MaxPrice: req.MaxPrice,
		Category: req.Category,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, 0, status.Error(codes.Internal, "db error")
	}
	return toProtoList(products), total, nil
}

// ── 转换函数 ──────────────────────────────────────────────────

func toProto(p repository.Product) *commonpb.Product {
	return &commonpb.Product{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price: &commonpb.Money{
			Amount:   p.AmountFen,
			Currency: p.Currency,
		},
		Category:   p.Category,
		ImageUrl:   p.ImageURL,
		Stock:      p.Stock,
		Rating:     p.Rating,
		SalesCount: p.SalesCount,
	}
}

func toProtoList(products []repository.Product) []*commonpb.Product {
	result := make([]*commonpb.Product, 0, len(products))
	for _, p := range products {
		result = append(result, toProto(p))
	}
	return result
}
