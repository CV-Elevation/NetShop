package service

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

type SearchQuery struct {
	Keyword  string `json:"keyword"`
	MaxPrice int64  `json:"max_price"`
	Category string `json:"category"`
}

type QueryExtractor struct {
	httpClient *http.Client
	baseURL    string
	model      string
	enabled    bool
}

func NewQueryExtractorFromEnv() *QueryExtractor {
	baseURL := strings.TrimSpace(os.Getenv("PRODUCT_QUERY_LLM_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := strings.TrimSpace(os.Getenv("PRODUCT_QUERY_LLM_MODEL"))
	if model == "" {
		model = "qwen3.5:0.8b"
	}
	enabled := !strings.EqualFold(strings.TrimSpace(os.Getenv("PRODUCT_QUERY_LLM_ENABLED")), "false")

	return &QueryExtractor{
		httpClient: &http.Client{Timeout: 6 * time.Second},
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		enabled:    enabled,
	}
}

func (e *QueryExtractor) Extract(ctx context.Context, query string) (SearchQuery, error) {
	if !e.enabled {
		return SearchQuery{}, fmt.Errorf("query extractor disabled")
	}
	if strings.TrimSpace(query) == "" {
		return SearchQuery{}, fmt.Errorf("empty query")
	}

	prompt := `你是商品搜索参数提取器。请从用户输入中提取搜索参数，并只输出 JSON，不要输出解释。

字段定义：
- keyword: 商品关键词（例如“防水登山鞋”“蓝牙耳机”）
- max_price: 最高价格，单位是“分”。例如“300元以内”则输出 30000。
- category: 商品分类（如“角色模型”“音乐、CD和黑胶唱片”），没有则输出空字符串。

输出格式：
{"keyword":"...","max_price":0,"category":"..."}

用户输入：` + query

	body := map[string]any{
		"model":      e.model,
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
		return SearchQuery{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return SearchQuery{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return SearchQuery{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return SearchQuery{}, fmt.Errorf("extract query failed: %s, body=%s", resp.Status, strings.TrimSpace(string(bodyBytes)))
	}

	var parsed struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return SearchQuery{}, err
	}

	jsonText := extractJSON(parsed.Message.Content)
	var result SearchQuery
	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		return SearchQuery{}, err
	}
	result.Keyword = strings.TrimSpace(result.Keyword)
	result.Category = strings.TrimSpace(result.Category)
	return result, nil
}

func extractJSON(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end < start {
		return text
	}
	return text[start : end+1]
}
