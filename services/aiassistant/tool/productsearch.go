package tool

import (
	"context"
	"encoding/json"
	"kuoz/netshop/platform/shared/proto/product"
	"strings"
	"unicode"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

var questionStopWords = map[string]struct{}{
	"请":      {},
	"帮我":     {},
	"我想":     {},
	"我要":     {},
	"想":      {},
	"查询":     {},
	"查":      {},
	"搜索":     {},
	"找":      {},
	"推荐":     {},
	"商品":     {},
	"产品":     {},
	"一下":     {},
	"有":      {},
	"and":    {},
	"or":     {},
	"the":    {},
	"a":      {},
	"an":     {},
	"please": {},
	"find":   {},
	"search": {},
}

type ProductSearchTool struct {
	productClient product.ProductServiceClient
}

func NewProductSearchTool(productClient product.ProductServiceClient) *ProductSearchTool {
	return &ProductSearchTool{
		productClient: productClient,
	}
}

type ProductSearchToolParam struct {
	Question string `json:"question"`
	MinPrice int    `json:"minprice"`
	MaxPrice int    `json:"maxprice"`
}

func (t *ProductSearchTool) ToolName() AgentTool {
	return AgentToolProductSearch
}

func (t *ProductSearchTool) Info() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
		Name:        string(AgentToolProductSearch),
		Description: openai.String("search products by price,name and description"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{
					"type":        "string",
					"description": "question containing keyword of products",
				},
				"minprice": map[string]any{
					"type":        "integer",
					"description": "Minimum Product Price",
				},
				"maxprice": map[string]any{
					"type":        "integer",
					"description": "Maximum Product Price",
				},
			},
			"required": []string{"question"},
		},
	})
}

func (t *ProductSearchTool) Execute(ctx context.Context, argumentsInJSON string) (string, error) {
	p := ProductSearchToolParam{}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", err
	}
	question := strings.ToLower(strings.TrimSpace(p.Question))
	minprice := p.MinPrice
	maxprice := p.MaxPrice
	if question == "" {
		return "请告诉我您想查询的商品信息", nil
	}
	// 从 question 中提取关键词，供商品搜索使用。
	keywords := extractKeywords(question)
	if keywords == "" {
		keywords = question
	}
	//根据关键词、最大值、最小值调用商品服务
	resp, err := t.productClient.SearchProducts(ctx, &product.SearchProductsRequest{Keyword: keywords, MaxPrice: int64(maxprice), MinPrice: int64(minprice)})

	if err != nil {
		return "", err
	}
	var builder strings.Builder
	for _, product := range resp.Items {
		builder.WriteString(product.Name)
	}
	return builder.String(), nil
}

func extractKeywords(question string) string {
	replacer := strings.NewReplacer(
		"，", " ", "。", " ", "？", " ", "！", " ", "；", " ", "：", " ", "、", " ",
		",", " ", ".", " ", "?", " ", "!", " ", ";", " ", ":", " ", "/", " ",
		"(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ", "\"", " ", "'", " ",
	)
	normalized := replacer.Replace(strings.ToLower(strings.TrimSpace(question)))

	tokens := strings.Fields(normalized)
	if len(tokens) == 0 {
		return ""
	}

	filtered := make([]string, 0, len(tokens))
	seen := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		token = cleanToken(token)
		if token == "" {
			continue
		}
		if _, ok := questionStopWords[token]; ok {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		filtered = append(filtered, token)
		if len(filtered) == 4 {
			break
		}
	}

	return strings.Join(filtered, " ")
}

func cleanToken(token string) string {
	var b strings.Builder
	b.Grow(len(token))
	for _, r := range token {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Han, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
