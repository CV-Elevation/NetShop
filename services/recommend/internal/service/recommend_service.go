package service

import (
	"context"

	commonpb "kuoz/netshop/platform/shared/proto/common"
	recommendpb "kuoz/netshop/platform/shared/proto/recommend"
)

type RecommendService struct{}

func NewRecommendService() *RecommendService {
	return &RecommendService{}
}

func (s *RecommendService) GetRecommendations(ctx context.Context, req *recommendpb.RecommendRequest) ([]*commonpb.Product, string, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	all := staticProducts()
	if int(limit) < len(all) {
		all = all[:limit]
	}
	return all, "static", nil
}

func (s *RecommendService) RecordBehavior(ctx context.Context, req *recommendpb.BehaviorEvent) error {
	return nil
}

func staticProducts() []*commonpb.Product {
	return []*commonpb.Product{
		{
			Id:       "c1834635-4cfc-4d30-a50d-de27277d5069",
			Name:     "狼与香辛料OST",
			Category: "音乐、CD和黑胶唱片",
			Price:    &commonpb.Money{Amount: 27491, Currency: "CNY"},
			ImageUrl: "https://m.media-amazon.com/images/I/51ZsKiAloxL._SX425_.jpg",
			Rating:   4.5,
		},
		{
			Id:       "c4f19c66-538e-4044-8624-b7db6a7c6fcf",
			Name:     "Good Smile Company Pop Up Parade Cyberpunk Edge Runners Lucy L Size Non-Scale",
			Category: "角色模型",
			Price:    &commonpb.Money{Amount: 110155, Currency: "CNY"},
			ImageUrl: "https://m.media-amazon.com/images/I/414WJV6AFjL._AC_SY879_.jpg",
			Rating:   4.5,
		},
	}
}
