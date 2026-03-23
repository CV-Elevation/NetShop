package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// GetByID 按 ID 查询单个商品
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (Product, bool, error) {
	var p Product
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, price_fen, currency, category, image_url, stock, rating, sales_count
		FROM products.items
		WHERE id = $1
	`, id).Scan(
		&p.ID, &p.Name, &p.Description,
		&p.AmountFen, &p.Currency,
		&p.Category, &p.ImageURL,
		&p.Stock, &p.Rating, &p.SalesCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, false, nil
	}
	if err != nil {
		return Product{}, false, err
	}
	return p, true, nil
}

// List 查询商品列表，支持分类过滤和分页
func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]Product, int32, error) {
	// ── 动态拼 WHERE ──────────────────────────────────────────
	where := []string{}
	args := []any{}
	idx := 1

	if filter.Category != "" {
		where = append(where, "category = $"+itoa(idx))
		args = append(args, filter.Category)
		idx++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// ── 查总数 ────────────────────────────────────────────────
	var total int32
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM products.items "+whereClause,
		args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// ── 查数据 ────────────────────────────────────────────────
	offset := (filter.Page - 1) * filter.PageSize
	args = append(args, filter.PageSize, offset)

	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, price_fen, currency, category, image_url, stock, rating, sales_count
		FROM products.items
		`+whereClause+`
		ORDER BY sales_count DESC
		LIMIT $`+itoa(idx)+` OFFSET $`+itoa(idx+1),
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return scanProducts(rows)
}

// Search 关键词 + 价格上限 + 分类过滤
func (r *PostgresRepository) Search(ctx context.Context, filter SearchFilter) ([]Product, int32, error) {
	where := []string{}
	args := []any{}
	idx := 1

	if filter.Keyword != "" {
		where = append(where, "(name ILIKE $"+itoa(idx)+" OR description ILIKE $"+itoa(idx)+")")
		args = append(args, "%"+filter.Keyword+"%")
		idx++
	}
	if filter.MaxPrice > 0 {
		where = append(where, "price_fen <= $"+itoa(idx))
		args = append(args, filter.MaxPrice)
		idx++
	}
	if filter.Category != "" {
		where = append(where, "category = $"+itoa(idx))
		args = append(args, filter.Category)
		idx++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int32
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM products.items "+whereClause,
		args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	args = append(args, filter.PageSize, offset)

	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, price_fen, currency, category, image_url, stock, rating, sales_count
		FROM products.items
		`+whereClause+`
		ORDER BY sales_count DESC
		LIMIT $`+itoa(idx)+` OFFSET $`+itoa(idx+1),
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return scanProducts(rows)
}

// ── 工具函数 ──────────────────────────────────────────────────

func scanProducts(rows pgx.Rows) ([]Product, int32, error) {
	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description,
			&p.AmountFen, &p.Currency,
			&p.Category, &p.ImageURL,
			&p.Stock, &p.Rating, &p.SalesCount,
		); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return products, int32(len(products)), nil
}

func itoa(i int) string {
	return strings.TrimSpace(strings.Replace("         "+string(rune('0'+i)), " ", "", -1))
}
