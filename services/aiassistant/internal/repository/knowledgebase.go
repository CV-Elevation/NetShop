package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	commonpb "kuoz/netshop/platform/shared/proto/common"
	productpb "kuoz/netshop/platform/shared/proto/product"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	SearchProducts(ctx context.Context, keyword string, limit int32) ([]*commonpb.Product, error)
	RetrieveKnowledge(ctx context.Context, question string, limit int32) ([]KnowledgeChunk, error)
}

type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type KnowledgeChunk struct {
	Content string
	Source  string
	Score   float64
}

type ProductRepository struct {
	productClient productpb.ProductServiceClient
	kb            *KnowledgeBaseRepository
}

func NewProductRepository(productClient productpb.ProductServiceClient, kb *KnowledgeBaseRepository) *ProductRepository {
	return &ProductRepository{productClient: productClient, kb: kb}
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

func (r *ProductRepository) BuildKnowledgeBase(ctx context.Context) error {
	if r.kb == nil {
		return nil
	}
	return r.kb.BuildKnowledgeBase(ctx)
}

func (r *ProductRepository) Close() {
	if r.kb != nil {
		r.kb.Close()
	}
}

func (r *ProductRepository) RetrieveKnowledge(ctx context.Context, question string, limit int32) ([]KnowledgeChunk, error) {
	if r.kb == nil {
		fallback, ok := fallbackFAQ(question)
		if !ok {
			return nil, nil
		}
		return []KnowledgeChunk{{
			Content: fallback,
			Source:  "fallback",
			Score:   0.6,
		}}, nil
	}
	if limit <= 0 {
		limit = 4
	}
	return r.kb.Query(ctx, question, int(limit))
}

type KnowledgeBaseRepository struct {
	pool     *pgxpool.Pool
	embedder EmbeddingProvider
}

func NewKnowledgeBaseRepository(ctx context.Context, dsn string, embedder EmbeddingProvider) (*KnowledgeBaseRepository, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("knowledge db dsn is empty")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	repo := &KnowledgeBaseRepository{pool: pool, embedder: embedder}
	if err := repo.ensureSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return repo, nil
}

func (r *KnowledgeBaseRepository) Close() {
	if r.pool != nil {
		r.pool.Close()
	}
}

func (r *KnowledgeBaseRepository) ensureSchema(ctx context.Context) error {
	const sql = `
CREATE SCHEMA IF NOT EXISTS knowledge;

CREATE TABLE IF NOT EXISTS knowledge.chunks (
    id         BIGSERIAL PRIMARY KEY,
    question   TEXT NOT NULL,
    answer     TEXT NOT NULL,
    chunk_text TEXT NOT NULL,
    source     TEXT NOT NULL DEFAULT 'seed',
    embedding  vector NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_knowledge_source ON knowledge.chunks(source);
`
	_, err := r.pool.Exec(ctx, sql)
	return err
}

func (r *KnowledgeBaseRepository) BuildKnowledgeBase(ctx context.Context) error {
	if r.embedder == nil {
		return nil
	}

	var count int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM knowledge.chunks").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	for _, item := range defaultFAQSeeds() {
		chunkText := fmt.Sprintf("问题: %s\n回答: %s", item.Question, item.Answer)
		embedding, err := r.embedder.Embed(ctx, chunkText)
		if err != nil {
			return err
		}
		if _, err := r.pool.Exec(ctx,
			`INSERT INTO knowledge.chunks (question, answer, chunk_text, source, embedding) VALUES ($1, $2, $3, $4, $5::vector)`,
			item.Question,
			item.Answer,
			chunkText,
			item.Source,
			vectorLiteral(embedding),
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *KnowledgeBaseRepository) Query(ctx context.Context, question string, limit int) ([]KnowledgeChunk, error) {
	if limit <= 0 {
		limit = 4
	}

	if r.embedder == nil {
		fallback, ok := fallbackFAQ(question)
		if !ok {
			return nil, nil
		}
		return []KnowledgeChunk{{Content: fallback, Source: "fallback", Score: 0.6}}, nil
	}

	embedding, err := r.embedder.Embed(ctx, question)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT chunk_text, source, 1 - (embedding <=> $1::vector) AS score
		 FROM knowledge.chunks
		 ORDER BY embedding <=> $1::vector
		 LIMIT $2`,
		vectorLiteral(embedding),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chunks := make([]KnowledgeChunk, 0, limit)
	for rows.Next() {
		var chunk KnowledgeChunk
		if err := rows.Scan(&chunk.Content, &chunk.Source, &chunk.Score); err != nil {
			return nil, err
		}
		if chunk.Score < 0.35 {
			continue
		}
		chunks = append(chunks, chunk)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(chunks) == 0 {
		fallback, ok := fallbackFAQ(question)
		if !ok {
			return nil, nil
		}
		return []KnowledgeChunk{{Content: fallback, Source: "fallback", Score: 0.5}}, nil
	}

	return chunks, nil
}

type faqSeed struct {
	Question string
	Answer   string
	Source   string
}

func defaultFAQSeeds() []faqSeed {
	return []faqSeed{
		{Question: "如何申请退款", Answer: "订单签收后 7 天内支持无理由退货，退款审核通过后 1-3 个工作日原路退回。", Source: "seed-refund"},
		{Question: "多久发货", Answer: "现货商品通常 24 小时内发货，可在订单详情页查看物流单号和轨迹。", Source: "seed-logistics"},
		{Question: "支持哪些支付方式", Answer: "支持主流线上支付方式，支付失败可稍后重试或更换支付渠道。", Source: "seed-payment"},
		{Question: "运费怎么计算", Answer: "运费会在结算页按收货地址和商品重量自动计算并展示。", Source: "seed-shipping-fee"},
	}
}

func fallbackFAQ(question string) (string, bool) {
	q := strings.ToLower(question)

	switch {
	case strings.Contains(q, "退款") || strings.Contains(q, "退货"):
		return "订单签收后 7 天内支持无理由退货；退款会在审核通过后 1-3 个工作日原路退回。", true
	case strings.Contains(q, "发货") || strings.Contains(q, "物流") || strings.Contains(q, "配送"):
		return "现货商品通常 24 小时内发货；可在订单详情页查看物流单号和实时轨迹。", true
	case strings.Contains(q, "支付"):
		return "当前支持主流线上支付方式，如支付失败可稍后重试或更换支付渠道。", true
	case strings.Contains(q, "运费"):
		return "运费会在结算页根据收货地址和商品重量自动计算并展示。", true
	default:
		return "", false
	}
}

func vectorLiteral(vector []float64) string {
	b := strings.Builder{}
	b.WriteString("[")
	for i, value := range vector {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
	}
	b.WriteString("]")
	return b.String()
}
