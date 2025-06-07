package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"alarket/internal/domain/entities"
	"alarket/internal/domain/repositories"
)

type TradeRepository struct {
	db *sql.DB
}

func NewTradeRepository(db *sql.DB) repositories.TradeRepository {
	return &TradeRepository{db: db}
}

func (r *TradeRepository) Save(ctx context.Context, trade *entities.Trade) error {
	query := `
		INSERT INTO trades (
			id, symbol, price, quantity, trade_time, is_buyer_market_maker, event_time
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		trade.ID,
		trade.Symbol,
		trade.Price,
		trade.Quantity,
		trade.Time,
		trade.IsBuyerMaker,
		trade.EventTime,
	)

	if err != nil {
		return fmt.Errorf("failed to save trade: %w", err)
	}

	return nil
}

func (r *TradeRepository) SaveBatch(ctx context.Context, trades []*entities.Trade) error {
	if len(trades) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	batch, err := tx.Prepare(`
		INSERT INTO trades (
			id, symbol, price, quantity, trade_time, is_buyer_market_maker, event_time
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}
	defer batch.Close()

	for _, trade := range trades {
		_, err := batch.Exec(
			trade.ID,
			trade.Symbol,
			trade.Price,
			trade.Quantity,
			trade.Time,
			trade.IsBuyerMaker,
			trade.EventTime,
		)
		if err != nil {
			return fmt.Errorf("failed to add trade to batch %s: %w", trade.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	return nil
}

func (r *TradeRepository) GetBySymbol(ctx context.Context, symbol string, from, to time.Time) ([]*entities.Trade, error) {
	query := `
		SELECT id, symbol, price, quantity, trade_time, is_buyer_market_maker, event_time
		FROM trades
		WHERE symbol = ? AND trade_time >= ? AND trade_time <= ?
		ORDER BY trade_time
	`

	rows, err := r.db.QueryContext(ctx, query, symbol, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	var trades []*entities.Trade
	for rows.Next() {
		var trade entities.Trade
		err := rows.Scan(
			&trade.ID,
			&trade.Symbol,
			&trade.Price,
			&trade.Quantity,
			&trade.Time,
			&trade.IsBuyerMaker,
			&trade.EventTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}
		trades = append(trades, &trade)
	}

	return trades, nil
}

func (r *TradeRepository) GetByID(ctx context.Context, id string) (*entities.Trade, error) {
	query := `
		SELECT id, symbol, price, quantity, trade_time, is_buyer_market_maker, event_time
		FROM trades
		WHERE id = ?
		LIMIT 1
	`

	var trade entities.Trade
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&trade.ID,
		&trade.Symbol,
		&trade.Price,
		&trade.Quantity,
		&trade.Time,
		&trade.IsBuyerMaker,
		&trade.EventTime,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get trade by id: %w", err)
	}

	return &trade, nil
}
