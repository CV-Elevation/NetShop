package repository

import (
	"context"
	"strings"

	commonpb "kuoz/netshop/platform/shared/proto/common"
	productpb "kuoz/netshop/platform/shared/proto/product"
)

type Repository interface {
	SearchProducts(ctx context.Context, keyword string, limit int32) ([]*commonpb.Product, error)
	QueryFAQ(ctx context.Context, question string) (string, bool, error)
}

type ProductRepository struct {
	productClient productpb.ProductServiceClient
}

func NewProductRepository(productClient productpb.ProductServiceClient) *ProductRepository {
	return &ProductRepository{productClient: productClient}
}

func (r *ProductRepository) SearchProducts(ctx context.Context, keyword string, limit int32) ([]*commonpb.Product, error) {
	if limit <= 0 {
		limit = 5
	}

	resp, err := r.productClient.SearchProducts(ctx, &productpb.SearchProductsRequest{
		Keyword: strings.TrimSpace(keyword),
		Pagination: &commonpb.Pagination{
			Page:     1,
			PageSize: limit,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.GetItems(), nil
}

func (r *ProductRepository) QueryFAQ(_ context.Context, question string) (string, bool, error) {
	q := strings.ToLower(question)

	switch {
	case strings.Contains(q, "退款") || strings.Contains(q, "退货"):
		return "订单签收后 7 天内支持无理由退货；退款会在审核通过后 1-3 个工作日原路退回。", true, nil
	case strings.Contains(q, "发货") || strings.Contains(q, "物流") || strings.Contains(q, "配送"):
		return "现货商品通常 24 小时内发货；可在订单详情页查看物流单号和实时轨迹。", true, nil
	case strings.Contains(q, "支付"):
		return "当前支持主流线上支付方式，如支付失败可稍后重试或更换支付渠道。", true, nil
	case strings.Contains(q, "运费"):
		return "运费会在结算页根据收货地址和商品重量自动计算并展示。", true, nil
	default:
		return "", false, nil
	}
}
