package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"alarket/internal/domain/entities"
	"alarket/internal/domain/repositories"
)

type BookTickerRepository struct {
	db *sql.DB
}

func NewBookTickerRepository(db *sql.DB) repositories.BookTickerRepository {
	return &BookTickerRepository{db: db}
}

func (r *BookTickerRepository) Save(ctx context.Context, ticker *entities.BookTicker) error {
	query := `
		INSERT INTO book_tickers (
			update_id, symbol, best_bid_price, best_bid_quantity,
			best_ask_price, best_ask_quantity, transaction_time, event_time
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		ticker.UpdateID,
		ticker.Symbol,
		ticker.BestBidPrice,
		ticker.BestBidQuantity,
		ticker.BestAskPrice,
		ticker.BestAskQuantity,
		ticker.TransactionTime,
		ticker.EventTime,
	)

	if err != nil {
		return fmt.Errorf("failed to save book ticker: %w", err)
	}

	return nil
}

func (r *BookTickerRepository) SaveBatch(ctx context.Context, tickers []*entities.BookTicker) error {
	if len(tickers) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	batch, err := tx.Prepare(`
		INSERT INTO book_tickers (
			update_id, symbol, best_bid_price, best_bid_quantity,
			best_ask_price, best_ask_quantity, transaction_time, event_time
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}
	defer batch.Close()

	for _, ticker := range tickers {
		_, err := batch.Exec(
			ticker.UpdateID,
			ticker.Symbol,
			ticker.BestBidPrice,
			ticker.BestBidQuantity,
			ticker.BestAskPrice,
			ticker.BestAskQuantity,
			ticker.TransactionTime,
			ticker.EventTime,
		)
		if err != nil {
			return fmt.Errorf("failed to add book ticker to batch for %s: %w", ticker.Symbol, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	return nil
}

func (r *BookTickerRepository) GetLatestBySymbol(ctx context.Context, symbol string) (*entities.BookTicker, error) {
	query := `
		SELECT update_id, symbol, best_bid_price, best_bid_quantity,
			   best_ask_price, best_ask_quantity, transaction_time, event_time
		FROM book_tickers
		WHERE symbol = ?
		ORDER BY event_time DESC
		LIMIT 1
	`

	var ticker entities.BookTicker
	err := r.db.QueryRowContext(ctx, query, symbol).Scan(
		&ticker.UpdateID,
		&ticker.Symbol,
		&ticker.BestBidPrice,
		&ticker.BestBidQuantity,
		&ticker.BestAskPrice,
		&ticker.BestAskQuantity,
		&ticker.TransactionTime,
		&ticker.EventTime,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest book ticker: %w", err)
	}

	return &ticker, nil
}

func (r *BookTickerRepository) GetBySymbol(ctx context.Context, symbol string, from, to time.Time) ([]*entities.BookTicker, error) {
	query := `
		SELECT update_id, symbol, best_bid_price, best_bid_quantity,
			   best_ask_price, best_ask_quantity, transaction_time, event_time
		FROM book_tickers
		WHERE symbol = ? AND event_time >= ? AND event_time <= ?
		ORDER BY event_time
	`

	rows, err := r.db.QueryContext(ctx, query, symbol, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query book tickers: %w", err)
	}
	defer rows.Close()

	var tickers []*entities.BookTicker
	for rows.Next() {
		var ticker entities.BookTicker
		err := rows.Scan(
			&ticker.UpdateID,
			&ticker.Symbol,
			&ticker.BestBidPrice,
			&ticker.BestBidQuantity,
			&ticker.BestAskPrice,
			&ticker.BestAskQuantity,
			&ticker.TransactionTime,
			&ticker.EventTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan book ticker: %w", err)
		}
		tickers = append(tickers, &ticker)
	}

	return tickers, nil
}
