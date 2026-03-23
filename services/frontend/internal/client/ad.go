package client

import (
	"context"

	adpb "kuoz/netshop/platform/shared/proto/ad"
)

type AdServiceClient struct {
	grpcClient adpb.AdServiceClient
}

type GetAdsRequest struct {
	UserID string
	Slot   adpb.AdSlot
	Limit  int32
}

func NewAdServiceClient(grpcClient adpb.AdServiceClient) *AdServiceClient {
	return &AdServiceClient{grpcClient: grpcClient}
}

func (c *AdServiceClient) GetAds(ctx context.Context, req GetAdsRequest) ([]*adpb.Ad, error) {
	resp, err := c.grpcClient.GetAds(ctx, &adpb.GetAdsRequest{
		UserId: req.UserID,
		Slot:   req.Slot,
		Limit:  req.Limit,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetItems(), nil
}
