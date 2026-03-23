package service

import (
	"context"

	adpb "kuoz/netshop/platform/shared/proto/ad"
)

type AdService struct{}

func NewAdService() *AdService {
	return &AdService{}
}

func (s *AdService) GetAds(ctx context.Context, req *adpb.GetAdsRequest) ([]*adpb.Ad, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 3
	}

	all := staticAds(req.Slot)
	if int(limit) < len(all) {
		all = all[:limit]
	}
	return all, nil
}

func (s *AdService) RecordAdEvent(ctx context.Context, req *adpb.AdEvent) error {
	return nil
}

func staticAds(slot adpb.AdSlot) []*adpb.Ad {
	return []*adpb.Ad{
		{
			Id:        "ad-001",
			Title:     "限时特惠，全场八折",
			ImageUrl:  "https://example.com/ad1.jpg",
			TargetUrl: "https://example.com/sale",
			Slot:      slot,
		},
		{
			Id:        "ad-002",
			Title:     "新品上市，立即抢购",
			ImageUrl:  "https://example.com/ad2.jpg",
			TargetUrl: "https://example.com/new",
			Slot:      slot,
		},
	}
}
