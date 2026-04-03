package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type AnswerGenerator interface {
	GenerateCustomerReply(ctx context.Context, question string, knowledgeContext string) (string, error)
}

type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type DoubaoClient struct {
	httpClient       *http.Client
	embeddingClient  *http.Client
	baseURL          string
	apiKey           string
	embeddingBaseURL string
	embeddingModel   string
	generationModel  string
}

func NewDoubaoClientFromEnv() *DoubaoClient {
	baseURL := strings.TrimSpace(os.Getenv("DOUBAO_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3"
	}
	embeddingBaseURL := strings.TrimSpace(os.Getenv("LOCAL_EMBEDDING_MODEL_URL"))
	if embeddingBaseURL == "" {
		embeddingBaseURL = "http://localhost:11434/api/embeddings"
	} else {
		embeddingBaseURL = normalizeEmbeddingEndpoint(embeddingBaseURL)
	}
	return &DoubaoClient{
		httpClient:       &http.Client{Timeout: 12 * time.Second},
		embeddingClient:  &http.Client{Timeout: 12 * time.Second},
		baseURL:          strings.TrimRight(baseURL, "/"),
		apiKey:           strings.TrimSpace(os.Getenv("DOUBAO_API_KEY")),
		embeddingBaseURL: embeddingBaseURL,
		embeddingModel:   defaultValue(os.Getenv("LOCAL_EMBEDDING_MODEL_NAME"), "qwen3-embedding:0.6b"),
		generationModel:  defaultValue(os.Getenv("DOUBAO_CHAT_MODEL"), "doubao-seed-2-0-mini"),
	}
}

func (c *DoubaoClient) Embed(ctx context.Context, text string) ([]float64, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("embedding input is empty")
	}

	url := c.embeddingBaseURL
	body := map[string]any{
		"model":  c.embeddingModel,
		"prompt": text,
	}

	var parsed struct {
		Embedding  []float64   `json:"embedding"`
		Embeddings [][]float64 `json:"embeddings"`
		Data       []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := c.postEmbeddingJSON(ctx, url, body, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Embedding) > 0 {
		return parsed.Embedding, nil
	}
	if len(parsed.Embeddings) > 0 && len(parsed.Embeddings[0]) > 0 {
		return parsed.Embeddings[0], nil
	}
	if len(parsed.Data) > 0 && len(parsed.Data[0].Embedding) > 0 {
		return parsed.Data[0].Embedding, nil
	}
	if len(parsed.Data) == 0 && len(parsed.Embeddings) == 0 && len(parsed.Embedding) == 0 {
		return nil, errors.New("empty embedding from local model")
	}
	return nil, errors.New("unrecognized embedding response format")
}

func (c *DoubaoClient) GenerateCustomerReply(ctx context.Context, question string, knowledgeContext string) (string, error) {
	if c.apiKey == "" {
		return "", errors.New("DOUBAO_API_KEY is empty")
	}
	if strings.TrimSpace(question) == "" {
		return "", errors.New("question is empty")
	}

	url := c.baseURL + "/chat/completions"
	systemPrompt := "你是 NetShop 客服助手。严格基于知识库内容回答。若知识库不足以回答，明确说不知道并建议用户联系人工客服。回答简洁、礼貌、可执行。"
	userPrompt := fmt.Sprintf("用户问题:\n%s\n\n知识库片段:\n%s\n\n请生成最终答复。", question, strings.TrimSpace(knowledgeContext))

	body := map[string]any{
		"model": c.generationModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.2,
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := c.postJSON(ctx, url, body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("empty completion from doubao")
	}

	answer := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if answer == "" {
		return "", errors.New("empty completion text from doubao")
	}
	return answer, nil
}

func (c *DoubaoClient) postJSON(ctx context.Context, url string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("doubao request failed: %s, body=%s", resp.Status, strings.TrimSpace(string(bodyBytes)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func (c *DoubaoClient) postEmbeddingJSON(ctx context.Context, url string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.embeddingClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("local embedding request failed: %s, body=%s", resp.Status, strings.TrimSpace(string(bodyBytes)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func normalizeEmbeddingEndpoint(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return "http://localhost:11434/api/embeddings"
	}
	if strings.HasSuffix(trimmed, "/api/embeddings") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/api/embedding") {
		return strings.TrimSuffix(trimmed, "/api/embedding") + "/api/embeddings"
	}
	if strings.HasSuffix(trimmed, "/api/chat") {
		return strings.TrimSuffix(trimmed, "/api/chat") + "/api/embeddings"
	}
	if strings.HasSuffix(trimmed, "/api/generate") {
		return strings.TrimSuffix(trimmed, "/api/generate") + "/api/embeddings"
	}
	return strings.TrimRight(trimmed, "/") + "/api/embeddings"
}

func defaultValue(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
