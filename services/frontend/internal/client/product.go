package client

import (
	"context"

	commonpb "kuoz/netshop/platform/shared/proto/common"
	productpb "kuoz/netshop/platform/shared/proto/product"
)

type ProductServiceClient struct {
	grpcClient productpb.ProductServiceClient
}

type ListProductsRequest struct {
	Category string
	Page     int32
	PageSize int32
}

type ListProductsResponse struct {
	Items []*commonpb.Product
	Total int32
}

type GetProductRequest struct {
	ID string
}

func NewProductServiceClient(grpcClient productpb.ProductServiceClient) *ProductServiceClient {
	return &ProductServiceClient{grpcClient: grpcClient}
}

func (c *ProductServiceClient) ListProducts(ctx context.Context, req ListProductsRequest) (ListProductsResponse, error) {
	resp, err := c.grpcClient.ListProducts(ctx, &productpb.ListProductsRequest{
		Category: req.Category,
		Pagination: &commonpb.Pagination{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
	})
	if err != nil {
		return ListProductsResponse{}, err
	}

	return ListProductsResponse{
		Items: resp.GetItems(),
		Total: resp.GetTotal(),
	}, nil
}

func (c *ProductServiceClient) GetProduct(ctx context.Context, req GetProductRequest) (*commonpb.Product, error) {
	resp, err := c.grpcClient.GetProduct(ctx, &productpb.GetProductRequest{Id: req.ID})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
