package rag

import (
	"fmt"
	"strings"

	"netshop/services/aiassistant/internal/repository"
)

func BuildKnowledgeContext(chunks []repository.KnowledgeChunk, maxChars int) string {
	if len(chunks) == 0 {
		return ""
	}
	if maxChars <= 0 {
		maxChars = 1800
	}

	b := strings.Builder{}
	for i, chunk := range chunks {
		line := fmt.Sprintf("[%d|%.2f|%s] %s\n", i+1, chunk.Score, chunk.Source, chunk.Content)
		if b.Len()+len(line) > maxChars {
			break
		}
		b.WriteString(line)
	}
	return strings.TrimSpace(b.String())
}

func BuildFallbackCustomerAnswer(chunks []repository.KnowledgeChunk) string {
	if len(chunks) == 0 {
		return "我暂时没有命中可用的客服知识。建议你补充订单号、支付方式或物流关键词，我再帮你精确查询。"
	}
	return chunks[0].Content
}
