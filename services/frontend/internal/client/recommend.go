package client

import (
	"context"

	commonpb "kuoz/netshop/platform/shared/proto/common"
	recommendpb "kuoz/netshop/platform/shared/proto/recommend"
)

type RecommendServiceClient struct {
	grpcClient recommendpb.RecommendServiceClient
}

type GetRecommendationsRequest struct {
	UserID string
	Scene  recommendpb.Scene
	Limit  int32
}

type GetRecommendationsResponse struct {
	Items    []*commonpb.Product
	Strategy string
}

func NewRecommendServiceClient(grpcClient recommendpb.RecommendServiceClient) *RecommendServiceClient {
	return &RecommendServiceClient{grpcClient: grpcClient}
}

func (c *RecommendServiceClient) GetRecommendations(ctx context.Context, req GetRecommendationsRequest) (GetRecommendationsResponse, error) {
	resp, err := c.grpcClient.GetRecommendations(ctx, &recommendpb.RecommendRequest{
		UserId: req.UserID,
		Scene:  req.Scene,
		Limit:  req.Limit,
	})
	if err != nil {
		return GetRecommendationsResponse{}, err
	}

	return GetRecommendationsResponse{
		Items:    resp.GetItems(),
		Strategy: resp.GetStrategy(),
	}, nil
}
