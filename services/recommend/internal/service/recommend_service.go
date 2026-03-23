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
			Id:       "0f0f1f26-0a6a-493a-8c53-0348a098929f",
			Name:     "狼与香辛料OST",
			Category: "音乐、CD和黑胶唱片",
			Price:    &commonpb.Money{Amount: 27491, Currency: "CNY"},
			ImageUrl: "https://m.media-amazon.com/images/I/51ZsKiAloxL._SX425_.jpg",
			Rating:   4.5,
		},
	}
}
