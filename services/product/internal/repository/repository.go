package repository

import "context"

type Product struct {
	ID          string
	Name        string
	Description string
	AmountFen   int64
	Currency    string
	Category    string
	ImageURL    string
	Stock       int32
	Rating      float32
	SalesCount  int64
}

type ListFilter struct {
	Category string
	Page     int32
	PageSize int32
}

type SearchFilter struct {
	Keyword  string
	MaxPrice int64
	MinPrice int64
	Category string
	Page     int32
	PageSize int32
}

type Repository interface {
	GetByID(ctx context.Context, id string) (Product, bool, error)
	List(ctx context.Context, filter ListFilter) ([]Product, int32, error)
	Search(ctx context.Context, filter SearchFilter) ([]Product, int32, error)
}
