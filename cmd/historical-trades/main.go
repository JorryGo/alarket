package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/spf13/cobra"

	"alarket/internal/application/usecases"
	"alarket/internal/infrastructure/binance"
	"alarket/internal/infrastructure/clickhouse"
	"alarket/internal/infrastructure/config"
)

var (
	symbol string
	days   int
)

var rootCmd = &cobra.Command{
	Use:   "historical-trades",
	Short: "Fetch historical trades for a specific symbol",
	Long: `This tool fetches historical trade data from Binance for a specific symbol.
It checks existing data in the database and only fetches older trades as needed.`,
	RunE: runHistoricalTrades,
}

func init() {
	rootCmd.Flags().StringVarP(&symbol, "symbol", "s", "", "Trading pair symbol (e.g., BTCUSDT)")
	rootCmd.Flags().IntVarP(&days, "days", "d", 7, "Number of days of historical data to fetch")

	rootCmd.MarkFlagRequired("symbol")
}

func runHistoricalTrades(cmd *cobra.Command, args []string) error {
	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Setup logger
	logLevel := slog.LevelInfo
	switch cfg.App.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Setup database
	db, err := setupDatabase(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create repositories
	tradeRepository := clickhouse.NewTradeRepository(db)

	// Create services
	historicalDataService := binance.NewHistoricalTradesService(
		cfg.Binance.APIKey,
		cfg.Binance.SecretKey,
		cfg.Binance.UseTestnet,
		logger,
	)

	// Create use case
	fetchHistoricalTradesUseCase := usecases.NewFetchHistoricalTradesUseCase(
		tradeRepository,
		historicalDataService,
		logger,
	)

	// Normalize symbol
	symbol = strings.ToUpper(symbol)

	logger.Info("Starting historical trades collection",
		"symbol", symbol,
		"days", days)

	// Execute use case
	if err := fetchHistoricalTradesUseCase.Execute(ctx, symbol, days); err != nil {
		logger.Error("Failed to fetch historical trades", "error", err)
		return err
	}

	logger.Info("Historical trades collection completed successfully")
	return nil
}

func setupDatabase(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*sql.DB, error) {
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s?debug=%t",
		cfg.ClickHouse.Username,
		cfg.ClickHouse.Password,
		cfg.ClickHouse.Host,
		cfg.ClickHouse.Port,
		cfg.ClickHouse.Database,
		cfg.ClickHouse.Debug,
	)

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	migrator := clickhouse.NewMigrator(db, logger)
	if err := migrator.Migrate(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
