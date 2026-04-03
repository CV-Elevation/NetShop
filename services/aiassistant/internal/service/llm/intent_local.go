package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	aiassistantpb "kuoz/netshop/platform/shared/proto/aiassistant"
)

type Intent string

const (
	IntentProductSearch           Intent = "product_search"
	IntentCustomerService         Intent = "customer_service"
	IntentQueryProductPerformance Intent = "query_product_performance"
	IntentChitchat                Intent = "chitchat"
)

type IntentClassifier interface {
	Classify(ctx context.Context, message string, history []*aiassistantpb.Message) ([]Intent, error)
}

type LocalIntentClassifier struct {
	httpClient *http.Client
	endpoint   string
	model      string
}

func NewLocalIntentClassifierFromEnv() *LocalIntentClassifier {
	endpoint := strings.TrimSpace(os.Getenv("LOCAL_INTENT_MODEL_URL"))
	if endpoint == "" {
		endpoint = "http://localhost:11434/api/chat"
	} else {
		endpoint = normalizeIntentModelEndpoint(endpoint)
	}

	model := strings.TrimSpace(os.Getenv("LOCAL_INTENT_MODEL_NAME"))
	if model == "" {
		model = "qwen3.5:0.8b"
	}

	return &LocalIntentClassifier{
		httpClient: &http.Client{Timeout: 2 * time.Second},
		endpoint:   endpoint,
		model:      model,
	}
}

func (c *LocalIntentClassifier) Classify(ctx context.Context, message string, history []*aiassistantpb.Message) ([]Intent, error) {
	fallbackIntents := keywordIntents(message)

	// 允许通过配置直接关闭本地模型，便于开发和降级。
	if strings.EqualFold(strings.TrimSpace(os.Getenv("LOCAL_INTENT_MODEL_ENABLED")), "false") {
		return fallbackIntents, nil
	}
	prompt := buildIntentPrompt(message, history)
	intents, err := c.callLocalModel(ctx, prompt)
	if err != nil {
		return fallbackIntents, err
	}
	if len(intents) == 0 {
		return fallbackIntents, errors.New("empty intents from local model")
	}
	return intents, nil
}

func (c *LocalIntentClassifier) callLocalModel(ctx context.Context, prompt string) ([]Intent, error) {
	body := map[string]any{
		"model":      c.model,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
		"think":      false,
		"stream":     false,
		"keep_alive": "30m",
		"options": map[string]any{
			"temperature": 0.1,
			"num_predict": 64,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("local intent model status: %s", resp.Status)
	}

	var parsed struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	return parseIntentResponse(parsed.Message.Content)
}

func normalizeIntentModelEndpoint(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return "http://localhost:11434/api/chat"
	}
	if strings.HasSuffix(trimmed, "/api/chat") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/api/generate") {
		return strings.TrimSuffix(trimmed, "/api/generate") + "/api/chat"
	}
	return strings.TrimRight(trimmed, "/") + "/api/chat"
}

func buildIntentPrompt(message string, history []*aiassistantpb.Message) string {
	_ = history
	return `你是意图识别器。分析用户消息，给出用户消息中包含的意图。

意图类型：
- product_search：用户表现出购买意向、寻找特定商品或请求推荐。
- customer_service：涉及订单、退换货、物流、支付、补差价、质量投诉等具体“售后或交易流程”问题。
- query_product_performance：询问商品性能、参数、使用体验等问题，表现出对商品的兴趣但不直接表达购买意向。
- chitchat：包含问候、无意义闲聊。
规则：
- 判断是否包含查询商品的语义,如推荐、找找、有没有、想买、适合露营用的等,如果有则添加 product_search 意图
- 判断是否包含客服相关的语义,如订单、退货、物流、支付、售后、丢失包裹、购买的商品有质量问题等,如果有则添加 customer_service 意图
- 判断是否包含商品性能相关的语义，如询问商品参数、性能、使用体验等，但没有直接表达购买意向的，如果有则添加 query_product_performance 意图
- 判断是否包含问候、闲聊等无购买或客服意图的语义，如果有则添加 chitchat 意图
- 核心原则： 请独立判断用户消息的每一个子句。如果一句话前半部分在抱怨质量（客服），后半部分在求推荐（搜索），必须同时返回两个意图。

输出格式示例（只输出 JSON，不要有多余文字）：
{
  "intents": ["product_search", "customer_service"]
}
所有意图都不存在时输出：
{
  "intents": []
}
用户消息：` + message
}

func keywordIntents(message string) []Intent {
	input := strings.ToLower(strings.TrimSpace(message))
	if input == "" {
		return []Intent{IntentChitchat}
	}

	found := make([]Intent, 0, 3)

	productKeywords := []string{"推荐", "商品", "买", "购买", "价格", "多少钱", "库存", "销量", "找", "有没有"}
	for _, keyword := range productKeywords {
		if strings.Contains(input, keyword) {
			found = appendIntentIfMissing(found, IntentProductSearch)
			break
		}
	}

	serviceKeywords := []string{"退款", "退货", "发货", "物流", "配送", "支付", "运费", "售后", "订单", "投诉", "质量"}
	for _, keyword := range serviceKeywords {
		if strings.Contains(input, keyword) {
			found = appendIntentIfMissing(found, IntentCustomerService)
			break
		}
	}

	performanceKeywords := []string{"参数", "性能", "评测", "体验", "续航", "发热", "掉色", "过敏"}
	for _, keyword := range performanceKeywords {
		if strings.Contains(input, keyword) {
			found = appendIntentIfMissing(found, IntentQueryProductPerformance)
			break
		}
	}

	chatKeywords := []string{"你好", "嗨", "在吗", "谢谢", "你是谁", "你会什么", "哈哈"}
	for _, keyword := range chatKeywords {
		if strings.Contains(input, keyword) {
			found = appendIntentIfMissing(found, IntentChitchat)
			break
		}
	}

	if len(found) == 0 {
		return []Intent{IntentChitchat}
	}
	return found
}

func parseIntentResponse(text string) ([]Intent, error) {
	jsonText := extractJSON(text)

	var raw struct {
		Intents json.RawMessage `json:"intents"`
	}
	if err := json.Unmarshal([]byte(jsonText), &raw); err != nil {
		return nil, fmt.Errorf("parse intents json failed: %w", err)
	}

	intents := make([]string, 0)
	switch {
	case len(raw.Intents) == 0:
		return nil, errors.New("intents is empty")
	case raw.Intents[0] == '[':
		if err := json.Unmarshal(raw.Intents, &intents); err != nil {
			return nil, err
		}
	case raw.Intents[0] == '"':
		var one string
		if err := json.Unmarshal(raw.Intents, &one); err != nil {
			return nil, err
		}
		intents = []string{one}
	default:
		return nil, fmt.Errorf("unsupported intents format: %s", string(raw.Intents))
	}

	mapped := make([]Intent, 0, len(intents))
	for _, label := range intents {
		intent := Intent(strings.TrimSpace(label))
		switch intent {
		case IntentProductSearch, IntentCustomerService, IntentQueryProductPerformance, IntentChitchat:
			mapped = appendIntentIfMissing(mapped, intent)
		}
	}

	if len(mapped) == 0 {
		return nil, errors.New("no valid intents")
	}
	return mapped, nil
}

func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end < start {
		return s
	}
	return s[start : end+1]
}

func appendIntentIfMissing(intents []Intent, target Intent) []Intent {
	for _, item := range intents {
		if item == target {
			return intents
		}
	}
	return append(intents, target)
}
