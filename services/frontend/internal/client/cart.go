package client

import (
	"context"

	cartpb "kuoz/netshop/platform/shared/proto/cart"
	commonpb "kuoz/netshop/platform/shared/proto/common"
)

type CartServiceClient struct {
	grpcClient cartpb.CartServiceClient
}

type AddCartItemRequest struct {
	UserID    string
	ProductID string
	Quantity  int32
}

type AddCartItemResponse struct {
	Item       *cartpb.CartItem
	TotalItems int32
}

type GetCartRequest struct {
	UserID string
}

type GetCartResponse struct {
	Items      []*cartpb.CartItem
	TotalPrice *commonpb.Money
	TotalCount int32
	HasInvalid bool
}

func NewCartServiceClient(grpcClient cartpb.CartServiceClient) *CartServiceClient {
	return &CartServiceClient{grpcClient: grpcClient}
}

func (c *CartServiceClient) AddItem(ctx context.Context, req AddCartItemRequest) (AddCartItemResponse, error) {
	resp, err := c.grpcClient.AddItem(ctx, &cartpb.AddItemRequest{
		UserId:    req.UserID,
		ProductId: req.ProductID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		return AddCartItemResponse{}, err
	}

	return AddCartItemResponse{
		Item:       resp.GetItem(),
		TotalItems: resp.GetTotalItems(),
	}, nil
}

func (c *CartServiceClient) GetCart(ctx context.Context, req GetCartRequest) (GetCartResponse, error) {
	resp, err := c.grpcClient.GetCart(ctx, &cartpb.GetCartRequest{UserId: req.UserID})
	if err != nil {
		return GetCartResponse{}, err
	}

	return GetCartResponse{
		Items:      resp.GetItems(),
		TotalPrice: resp.GetTotalPrice(),
		TotalCount: resp.GetTotalCount(),
		HasInvalid: resp.GetHasInvalid(),
	}, nil
}
