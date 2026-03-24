package service

import (
	"context"
	"fmt"
	"strings"

	aiassistantpb "kuoz/netshop/platform/shared/proto/aiassistant"
	commonpb "kuoz/netshop/platform/shared/proto/common"
	"netshop/services/aiassistant/internal/repository"
	"netshop/services/aiassistant/internal/service/llm"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AIAssistantService struct {
	repo repository.Repository
}

func NewAIAssistantService(repo repository.Repository) *AIAssistantService {
	return &AIAssistantService{repo: repo}
}

func (s *AIAssistantService) Chat(ctx context.Context, req *aiassistantpb.ChatRequest) (*aiassistantpb.ChatResponse, error) {
	if req == nil || strings.TrimSpace(req.GetMessage()) == "" {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	result, err := s.buildChatResult(ctx, req)
	if err != nil {
		return nil, err
	}

	return &aiassistantpb.ChatResponse{
		Text:      result.Text,
		ToolCalls: result.ToolCalls,
		Products:  result.Products,
	}, nil
}

func (s *AIAssistantService) ChatStream(ctx context.Context, req *aiassistantpb.ChatRequest, send func(*aiassistantpb.ChatChunk) error) error {
	if send == nil {
		return status.Error(codes.Internal, "send callback is nil")
	}

	result, err := s.buildChatResult(ctx, req)
	if err != nil {
		return err
	}

	for _, toolCall := range result.ToolCalls {
		if err := send(&aiassistantpb.ChatChunk{
			ChunkType: "tool_status",
			ToolCall: &aiassistantpb.ToolCall{
				ToolName: toolCall.GetToolName(),
				Status:   "done",
				Summary:  toolCall.GetSummary(),
			},
		}); err != nil {
			return err
		}
	}

	for _, part := range splitText(result.Text, 18) {
		if err := send(&aiassistantpb.ChatChunk{
			ChunkType: "text",
			Delta:     part,
		}); err != nil {
			return err
		}
	}

	if len(result.Products) > 0 {
		if err := send(&aiassistantpb.ChatChunk{
			ChunkType: "products",
			Products:  result.Products,
		}); err != nil {
			return err
		}
	}

	return send(&aiassistantpb.ChatChunk{
		ChunkType: "done",
		Done:      true,
	})
}

type chatResult struct {
	Text      string
	ToolCalls []*aiassistantpb.ToolCall
	Products  []*commonpb.Product
}

func (s *AIAssistantService) buildChatResult(ctx context.Context, req *aiassistantpb.ChatRequest) (*chatResult, error) {
	message := strings.TrimSpace(req.GetMessage())
	input := strings.ToLower(message)

	result := &chatResult{
		ToolCalls: make([]*aiassistantpb.ToolCall, 0),
		Products:  make([]*commonpb.Product, 0),
	}

	if isProductQuery(input) {
		keyword := llm.ExtractProductKeyword(message)
		products, err := s.repo.SearchProducts(ctx, keyword, 5)
		if err != nil {
			return nil, status.Error(codes.Internal, "search products failed")
		}

		result.Products = products
		if len(result.Products) == 0 {
			result.Text = "我没有找到匹配的商品。你可以换个关键词，例如商品名、分类或用途。"
			result.ToolCalls = append(result.ToolCalls, &aiassistantpb.ToolCall{
				ToolName: "search_products",
				Status:   "done",
				Summary:  fmt.Sprintf("关键词 \"%s\" 未命中商品", keyword),
			})
			return result, nil
		}

		result.ToolCalls = append(result.ToolCalls, &aiassistantpb.ToolCall{
			ToolName: "search_products",
			Status:   "done",
			Summary:  fmt.Sprintf("已根据关键词 \"%s\" 检索到 %d 个商品", keyword, len(result.Products)),
		})
		result.Text = fmt.Sprintf("我为你找到 %d 个相关商品，已附在下方。", len(result.Products))
		return result, nil
	}

	if isFAQQuery(input) {
		answer, ok, err := s.repo.QueryFAQ(ctx, message)
		if err != nil {
			return nil, status.Error(codes.Internal, "query faq failed")
		}

		result.ToolCalls = append(result.ToolCalls, &aiassistantpb.ToolCall{
			ToolName: "query_faq",
			Status:   "done",
		})

		if !ok {
			result.ToolCalls[0].Summary = "暂无匹配 FAQ"
			result.Text = "这个问题我暂时没有命中 FAQ。你可以提供更具体的订单/支付/物流关键词。"
			return result, nil
		}

		result.ToolCalls[0].Summary = "已返回 FAQ 结果"
		result.Text = answer
		return result, nil
	}

	result.Text = "我可以帮你查商品推荐，或回答退款、物流、支付等常见问题。你可以直接告诉我需求。"
	return result, nil
}

func isProductQuery(input string) bool {
	keywords := []string{"推荐", "商品", "买", "购买", "price", "价格", "多少钱", "库存", "销量"}
	for _, keyword := range keywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func isFAQQuery(input string) bool {
	keywords := []string{"退款", "退货", "发货", "物流", "配送", "支付", "运费"}
	for _, keyword := range keywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func splitText(text string, chunkSize int) []string {
	if text == "" {
		return nil
	}
	runes := []rune(text)
	if chunkSize <= 0 || len(runes) <= chunkSize {
		return []string{text}
	}

	parts := make([]string, 0, len(runes)/chunkSize+1)
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		parts = append(parts, string(runes[start:end]))
	}
	return parts
}
