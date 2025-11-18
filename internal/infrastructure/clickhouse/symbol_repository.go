package clickhouse

import (
	"context"
	"database/sql"
	"fmt"

	"alarket/internal/domain/entities"
	"alarket/internal/domain/repositories"
)

type SymbolRepository struct {
	db             *sql.DB
	symbols        []*entities.Symbol // In-memory cache for now
	allowedSymbols map[string]bool    // Filter symbols (nil = all symbols)
}

func NewSymbolRepository(db *sql.DB, symbols []*entities.Symbol, allowedSymbols []string) repositories.SymbolRepository {
	// Create a map for fast lookup if symbols are specified
	var allowedMap map[string]bool
	if len(allowedSymbols) > 0 {
		allowedMap = make(map[string]bool, len(allowedSymbols))
		for _, symbol := range allowedSymbols {
			allowedMap[symbol] = true
		}
	}

	return &SymbolRepository{
		db:             db,
		symbols:        symbols,
		allowedSymbols: allowedMap,
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
		if !symbol.IsActive() {
			continue
		}

		// If allowedSymbols is set, filter by the list
		if r.allowedSymbols != nil {
			if r.allowedSymbols[symbol.Name] {
				active = append(active, symbol)
			}
		} else {
			// If allowedSymbols is nil (SYMBOLS not configured), return all active symbols
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
