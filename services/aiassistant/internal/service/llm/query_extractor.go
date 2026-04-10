package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ProductSearchQuery struct {
	Keyword  string
	MaxPrice int64
	MinPrice int64
	Category string
}

type ProductQueryExtractor struct {
	httpClient *http.Client
	baseURL    string
	model      string
	enabled    bool
}

func NewProductQueryExtractorFromEnv() *ProductQueryExtractor {
	baseURL := strings.TrimSpace(os.Getenv("PRODUCT_QUERY_LLM_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:11434/api/chat"
	} else {
		baseURL = normalizeQueryEndpoint(baseURL)
	}
	model := strings.TrimSpace(os.Getenv("PRODUCT_QUERY_LLM_MODEL"))
	if model == "" {
		model = "qwen3.5:0.8b"
	}
	enabled := !strings.EqualFold(strings.TrimSpace(os.Getenv("PRODUCT_QUERY_LLM_ENABLED")), "false")

	return &ProductQueryExtractor{
		httpClient: &http.Client{Timeout: 6 * time.Second},
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		enabled:    enabled,
	}
}

func (e *ProductQueryExtractor) Extract(ctx context.Context, message string) (ProductSearchQuery, error) {
	if !e.enabled {
		return ProductSearchQuery{}, fmt.Errorf("product query extractor disabled")
	}
	if strings.TrimSpace(message) == "" {
		return ProductSearchQuery{}, fmt.Errorf("empty message")
	}

	prompt := `你是商品搜索参数提取器。请从用户输入中提取商品搜索条件，只输出 JSON，不要输出任何解释。

字段定义：
- keyword: 商品关键词，尽量短，例如“蓝牙耳机”“防水登山鞋”“洗碗机”
- max_price: 最高价格，单位是“分”。例如“300元以内”输出 30000；未提到则输出 0
- min_price: 最低价格，单位是“分”。例如“1000元以上”输出 100000；未提到则输出 0
- category: 商品分类，没有就输出空字符串

判定规则：
- 如果用户说“300元以内”“不超过500”“500以下”，提取到 max_price
- 如果用户说“1000元以上”“至少2000”“起步价3000”，提取到 min_price
- 如果用户同时给出上下限，两个字段都填
- keyword 要去掉价格、分类等修饰词，只保留商品名或核心搜索词

输出格式：
{"keyword":"","max_price":0,"min_price":0,"category":""}

用户输入：` + message

	body := map[string]any{
		"model":      e.model,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
		"think":      false,
		"stream":     false,
		"keep_alive": "30m",
		"options": map[string]any{
			"temperature": 0.1,
			"num_predict": 96,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return ProductSearchQuery{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL, bytes.NewReader(payload))
	if err != nil {
		return ProductSearchQuery{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return ProductSearchQuery{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return ProductSearchQuery{}, fmt.Errorf("product query extraction failed: %s, body=%s", resp.Status, strings.TrimSpace(string(bodyBytes)))
	}

	var parsed struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return ProductSearchQuery{}, err
	}

	jsonText := extractQueryJSON(parsed.Message.Content)
	var result ProductSearchQuery
	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		return ProductSearchQuery{}, err
	}
	result.Keyword = strings.TrimSpace(result.Keyword)
	result.Category = strings.TrimSpace(result.Category)
	return result, nil
}

func extractQueryJSON(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end < start {
		return text
	}
	return text[start : end+1]
}

func normalizeQueryEndpoint(endpoint string) string {
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
