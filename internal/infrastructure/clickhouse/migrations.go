package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

type Migrator struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewMigrator(db *sql.DB, logger *slog.Logger) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger,
	}
}

func (m *Migrator) Migrate(ctx context.Context) error {
	migrations := []struct {
		name  string
		query string
	}{
		{
			name: "create_trades_table",
			query: `
				CREATE TABLE IF NOT EXISTS trades (
					id String,
					symbol String,
					price Float64,
					quantity Float64,
					buyer_order_id Int64,
					seller_order_id Int64,
					trade_time DateTime64(3),
					is_buyer_maker Bool,
					event_time DateTime64(3),
					created_at DateTime64(3) DEFAULT now64(3)
				)
				ENGINE = MergeTree()
				PARTITION BY toYYYYMM(trade_time)
				ORDER BY (symbol, trade_time, id)
				SETTINGS index_granularity = 8192
			`,
		},
		{
			name: "create_book_tickers_table",
			query: `
				CREATE TABLE IF NOT EXISTS book_tickers (
					update_id Int64,
					symbol String,
					best_bid_price Float64,
					best_bid_quantity Float64,
					best_ask_price Float64,
					best_ask_quantity Float64,
					transaction_time DateTime64(3),
					event_time DateTime64(3),
					created_at DateTime64(3) DEFAULT now64(3)
				)
				ENGINE = MergeTree()
				PARTITION BY toYYYYMM(event_time)
				ORDER BY (symbol, event_time, update_id)
				SETTINGS index_granularity = 8192
			`,
		},
	}

	for _, migration := range migrations {
		m.logger.Info("Running migration", "name", migration.name)
		if _, err := m.db.ExecContext(ctx, migration.query); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration.name, err)
		}
	}

	m.logger.Info("All migrations completed successfully")
	return nil
}
