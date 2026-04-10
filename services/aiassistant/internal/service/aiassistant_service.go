package service

import (
	"context"
	"fmt"
	"strings"

	aiassistantpb "kuoz/netshop/platform/shared/proto/aiassistant"
	commonpb "kuoz/netshop/platform/shared/proto/common"
	"netshop/services/aiassistant/internal/repository"
	"netshop/services/aiassistant/internal/service/llm"
	"netshop/services/aiassistant/internal/service/rag"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AIAssistantService struct {
	repo        repository.Repository
	intent      llm.IntentClassifier
	query       *llm.ProductQueryExtractor
	answerAgent llm.AnswerGenerator
}

func NewAIAssistantService(repo repository.Repository, intent llm.IntentClassifier, answerAgent llm.AnswerGenerator) *AIAssistantService {
	return &AIAssistantService{repo: repo, intent: intent, query: llm.NewProductQueryExtractorFromEnv(), answerAgent: answerAgent}
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
		status := toolCall.GetStatus()
		if status == "" {
			status = "done"
		}
		if err := send(&aiassistantpb.ChatChunk{
			ChunkType: "tool_status",
			ToolCall: &aiassistantpb.ToolCall{
				ToolName: toolCall.GetToolName(),
				Status:   status,
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

	result := &chatResult{
		ToolCalls: make([]*aiassistantpb.ToolCall, 0),
		Products:  make([]*commonpb.Product, 0),
	}

	intents := []llm.Intent{llm.IntentChitchat}
	if s.intent != nil {
		classifiedIntents, err := s.intent.Classify(ctx, message, req.GetHistory())
		if err == nil {
			if len(classifiedIntents) > 0 {
				intents = classifiedIntents
			}
		} else {
			result.ToolCalls = append(result.ToolCalls, &aiassistantpb.ToolCall{
				ToolName: "intent_classification",
				Status:   "error",
				Summary:  "本地意图模型不可用，已降级到规则判断",
			})
			intents = fallbackIntentsByRule(message)
		}
	}

	result.ToolCalls = append(result.ToolCalls, &aiassistantpb.ToolCall{
		ToolName: "intent_classification",
		Status:   "done",
		Summary:  fmt.Sprintf("意图=%v", intents),
	})

	hasProductIntent := containsIntent(intents, llm.IntentProductSearch)
	hasServiceIntent := containsIntent(intents, llm.IntentCustomerService)

	parts := make([]string, 0, 2)

	if hasServiceIntent {
		chunks, err := s.repo.RetrieveKnowledge(ctx, message, 4)
		if err != nil {
			return nil, status.Error(codes.Internal, "retrieve knowledge failed")
		}

		result.ToolCalls = append(result.ToolCalls, &aiassistantpb.ToolCall{
			ToolName: "retrieve_knowledge",
			Status:   "done",
			Summary:  fmt.Sprintf("命中 %d 条知识片段", len(chunks)),
		})

		if len(chunks) == 0 {
			parts = append(parts, "客服答复：这个问题我暂时没有命中 FAQ。你可以提供更具体的订单/支付/物流关键词。")
		} else {
			knowledgeContext := rag.BuildKnowledgeContext(chunks, 1800)
			if s.answerAgent == nil {
				parts = append(parts, "客服答复："+rag.BuildFallbackCustomerAnswer(chunks))
			} else {
				result.ToolCalls = append(result.ToolCalls, &aiassistantpb.ToolCall{
					ToolName: "generate_reply",
					Status:   "done",
				})

				answer, err := s.answerAgent.GenerateCustomerReply(ctx, message, knowledgeContext)
				if err != nil {
					result.ToolCalls[len(result.ToolCalls)-1].Status = "error"
					result.ToolCalls[len(result.ToolCalls)-1].Summary = fmt.Sprintf("Seed 模型调用失败，已返回检索摘要（原因：%v）", err)
					parts = append(parts, "客服答复："+rag.BuildFallbackCustomerAnswer(chunks))
				} else {
					result.ToolCalls[len(result.ToolCalls)-1].Summary = "已生成客服回复"
					parts = append(parts, "客服答复："+answer)
				}
			}
		}
	}

	if hasProductIntent {
		searchQuery := llm.ProductSearchQuery{Keyword: message}
		if s.query != nil {
			extractedQuery, err := s.query.Extract(ctx, message)
			if err != nil {
				result.ToolCalls = append(result.ToolCalls, &aiassistantpb.ToolCall{
					ToolName: "search_query_extraction",
					Status:   "error",
					Summary:  fmt.Sprintf("LLM 提取商品查询失败，已回退原始关键词（原因：%v）", err),
				})
			} else {
				searchQuery = extractedQuery
				result.ToolCalls = append(result.ToolCalls, &aiassistantpb.ToolCall{
					ToolName: "search_query_extraction",
					Status:   "done",
					Summary:  fmt.Sprintf("keyword=%q max_price=%d min_price=%d category=%q", searchQuery.Keyword, searchQuery.MaxPrice, searchQuery.MinPrice, searchQuery.Category),
				})
			}
		}

		products, err := s.repo.SearchProducts(ctx, searchQuery, 5)
		if err != nil {
			return nil, status.Error(codes.Internal, "search products failed")
		}

		result.Products = products
		if len(result.Products) == 0 {
			result.ToolCalls = append(result.ToolCalls, &aiassistantpb.ToolCall{
				ToolName: "search_products",
				Status:   "done",
				Summary:  fmt.Sprintf("关键词=%q max_price=%d min_price=%d category=%q 未命中商品", searchQuery.Keyword, searchQuery.MaxPrice, searchQuery.MinPrice, searchQuery.Category),
			})
			parts = append(parts, "商品检索：我没有找到匹配的商品。你可以换个关键词，例如商品名、分类或用途。")
		} else {
			result.ToolCalls = append(result.ToolCalls, &aiassistantpb.ToolCall{
				ToolName: "search_products",
				Status:   "done",
				Summary:  fmt.Sprintf("已根据 keyword=%q max_price=%d min_price=%d category=%q 检索到 %d 个商品", searchQuery.Keyword, searchQuery.MaxPrice, searchQuery.MinPrice, searchQuery.Category, len(result.Products)),
			})
			parts = append(parts, fmt.Sprintf("商品检索：我为你找到 %d 个相关商品，已附在下方。", len(result.Products)))
		}
	}

	if len(parts) > 0 {
		result.Text = strings.Join(parts, "\n\n")
		return result, nil
	}

	result.Text = "我可以帮你查商品推荐，或回答退款、物流、支付等常见问题。你可以直接告诉我需求。"
	return result, nil
}

func containsIntent(intents []llm.Intent, target llm.Intent) bool {
	for _, item := range intents {
		if item == target {
			return true
		}
	}
	return false
}

func fallbackIntentsByRule(message string) []llm.Intent {
	input := strings.ToLower(strings.TrimSpace(message))
	found := make([]llm.Intent, 0, 2)

	if isProductQuery(input) {
		found = append(found, llm.IntentProductSearch)
	}
	if isFAQQuery(input) {
		found = append(found, llm.IntentCustomerService)
	}
	if len(found) == 0 {
		return []llm.Intent{llm.IntentChitchat}
	}
	return found
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
