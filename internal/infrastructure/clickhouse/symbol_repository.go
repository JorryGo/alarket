package clickhouse

import (
	"context"
	"database/sql"
	"fmt"

	"alarket/internal/domain/entities"
	"alarket/internal/domain/repositories"
)

type SymbolRepository struct {
	db      *sql.DB
	symbols []*entities.Symbol // In-memory cache for now
}

func NewSymbolRepository(db *sql.DB, symbols []*entities.Symbol) repositories.SymbolRepository {
	return &SymbolRepository{
		db:      db,
		symbols: symbols,
	}
}

func (r *SymbolRepository) GetAll(ctx context.Context) ([]*entities.Symbol, error) {
	// For now, return from memory
	// In production, this would query from database
	return r.symbols, nil
}

func (r *SymbolRepository) GetActiveUsdt(ctx context.Context) ([]*entities.Symbol, error) {
	active := make([]*entities.Symbol, 0)
	for _, symbol := range r.symbols {
		if symbol.IsActive() && (symbol.BaseAsset == "USDT" || symbol.QuoteAsset == "USDT") {
			active = append(active, symbol)
		}
	}
	return active, nil
}

func (r *SymbolRepository) GetByName(ctx context.Context, name string) (*entities.Symbol, error) {
	for _, symbol := range r.symbols {
		if symbol.Name == name {
			return symbol, nil
		}
	}
	return nil, fmt.Errorf("symbol %s not found", name)
}

func (r *SymbolRepository) UpdateStatus(ctx context.Context, name string, status entities.SymbolStatus) error {
	for _, symbol := range r.symbols {
		if symbol.Name == name {
			symbol.Status = status
			return nil
		}
	}
	return fmt.Errorf("symbol %s not found", name)
}
