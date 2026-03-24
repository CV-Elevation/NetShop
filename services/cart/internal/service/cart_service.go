package service

import (
	"context"
	"log"

	cartpb "kuoz/netshop/platform/shared/proto/cart"
	commonpb "kuoz/netshop/platform/shared/proto/common"
	productpb "kuoz/netshop/platform/shared/proto/product"
	"netshop/services/cart/internal/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CartService struct {
	repo        repository.Repository
	productConn productpb.ProductServiceClient
}

func NewCartService(repo repository.Repository, productConn productpb.ProductServiceClient) *CartService {
	return &CartService{repo: repo, productConn: productConn}
}

func (s *CartService) AddItem(ctx context.Context, userID, productID string, quantity int32) (*cartpb.CartItem, int32, error) {
	if userID == "" {
		return nil, 0, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if productID == "" {
		return nil, 0, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if quantity <= 0 {
		return nil, 0, status.Error(codes.InvalidArgument, "quantity must be greater than 0")
	}

	// 查商品信息
	p, err := s.productConn.GetProduct(ctx, &productpb.GetProductRequest{Id: productID})
	if err != nil {
		return nil, 0, status.Error(codes.NotFound, "product not found")
	}

	// 累加数量
	newQty, err := s.repo.AddItem(ctx, userID, productID, quantity)
	if err != nil {
		return nil, 0, status.Error(codes.Internal, "redis error")
	}

	// 查购物车总条目数
	items, err := s.repo.GetItems(ctx, userID)
	if err != nil {
		return nil, 0, status.Error(codes.Internal, "redis error")
	}

	stockStatus, stockCount := resolveStockStatus(p.Stock, newQty)

	item := &cartpb.CartItem{
		ProductId:   productID,
		Name:        p.Name,
		Price:       p.Price,
		Quantity:    newQty,
		Subtotal:    &commonpb.Money{Amount: p.Price.Amount * int64(newQty), Currency: p.Price.Currency},
		ImageUrl:    p.ImageUrl,
		StockStatus: stockStatus,
		StockCount:  stockCount,
		Checked:     true,
	}

	return item, int32(len(items)), nil
}

func (s *CartService) GetCart(ctx context.Context, userID string) ([]*cartpb.CartItem, *commonpb.Money, int32, bool, error) {
	if userID == "" {
		return nil, nil, 0, false, status.Error(codes.InvalidArgument, "user_id is required")
	}

	items, err := s.repo.GetItems(ctx, userID)
	if err != nil {
		return nil, nil, 0, false, status.Error(codes.Internal, "redis error")
	}
	checked, err := s.repo.GetChecked(ctx, userID)
	if err != nil {
		return nil, nil, 0, false, status.Error(codes.Internal, "redis error")
	}

	var cartItems []*cartpb.CartItem
	var totalFen int64
	var totalCount int32
	hasInvalid := false
	currency := "CNY"

	for productID, quantity := range items {
		p, err := s.productConn.GetProduct(ctx, &productpb.GetProductRequest{Id: productID})
		if err != nil {
			log.Printf("[cart] product %s not found, skipping", productID)
			hasInvalid = true
			continue
		}

		stockStatus, stockCount := resolveStockStatus(p.Stock, quantity)
		if stockStatus == cartpb.StockStatus_INSUFFICIENT || stockStatus == cartpb.StockStatus_OUT_OF_STOCK {
			hasInvalid = true
		}

		isChecked := checked[productID]
		subtotal := p.Price.Amount * int64(quantity)
		if isChecked {
			totalFen += subtotal
			totalCount += quantity
		}
		currency = p.Price.Currency

		cartItems = append(cartItems, &cartpb.CartItem{
			ProductId:   productID,
			Name:        p.Name,
			Price:       p.Price,
			Quantity:    quantity,
			Subtotal:    &commonpb.Money{Amount: subtotal, Currency: p.Price.Currency},
			ImageUrl:    p.ImageUrl,
			StockStatus: stockStatus,
			StockCount:  stockCount,
			Checked:     isChecked,
		})
	}

	return cartItems, &commonpb.Money{Amount: totalFen, Currency: currency}, totalCount, hasInvalid, nil
}

func (s *CartService) ClearCart(ctx context.Context, userID string) error {
	if userID == "" {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}
	if err := s.repo.ClearCart(ctx, userID); err != nil {
		return status.Error(codes.Internal, "redis error")
	}
	return nil
}

// resolveStockStatus 根据库存和数量计算库存状态
func resolveStockStatus(stock int32, quantity int32) (cartpb.StockStatus, int32) {
	if stock == 0 {
		return cartpb.StockStatus_OUT_OF_STOCK, 0
	}
	if stock < quantity {
		return cartpb.StockStatus_INSUFFICIENT, stock
	}
	if stock < 10 {
		return cartpb.StockStatus_LOW_STOCK, stock
	}
	return cartpb.StockStatus_IN_STOCK, stock
}
