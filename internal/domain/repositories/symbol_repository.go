package repositories

import (
	"context"

	"alarket/internal/domain/entities"
)

type SymbolRepository interface {
	GetAll(ctx context.Context) ([]*entities.Symbol, error)
	GetActiveUsdt(ctx context.Context) ([]*entities.Symbol, error)
	GetByName(ctx context.Context, name string) (*entities.Symbol, error)
	UpdateStatus(ctx context.Context, name string, status entities.SymbolStatus) error
}
