package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/spf13/cobra"

	"alarket/internal/domain/entities"
	"alarket/internal/infrastructure/clickhouse"
	"alarket/internal/infrastructure/config"
)

var (
	filePath string
	symbol   string
)

var rootCmd = &cobra.Command{
	Use:   "file-import",
	Short: "Import trade data from CSV files",
	Long: `This tool imports trade data from CSV files into ClickHouse database.
It supports large files with streaming processing and batch saves.

CSV format expected:
ID,Price,Quantity,QuoteQuantity,Timestamp,IsBuyerMaker,IsBestMatch`,
	RunE: runFileImport,
}

func init() {
	rootCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to CSV file to import")
	rootCmd.Flags().StringVarP(&symbol, "symbol", "s", "", "Trading pair symbol (e.g., ETHUSDT)")

	rootCmd.MarkFlagRequired("file")
	rootCmd.MarkFlagRequired("symbol")
}

func runFileImport(cmd *cobra.Command, args []string) error {
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

	// Create repository
	tradeRepository := clickhouse.NewTradeRepository(db)

	// Normalize symbol
	symbol = strings.ToUpper(symbol)

	logger.Info("Starting CSV import",
		"file", filePath,
		"symbol", symbol)

	// Import CSV file
	if err := importCSVFile(ctx, filePath, symbol, tradeRepository.(*clickhouse.TradeRepository), logger); err != nil {
		logger.Error("Failed to import CSV file", "error", err)
		return err
	}

	logger.Info("CSV import completed successfully")
	return nil
}

func importCSVFile(ctx context.Context, filePath, symbol string, repository *clickhouse.TradeRepository, logger *slog.Logger) error {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file size for progress tracking
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// Create CSV reader
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 7 // Expect 7 fields per record

	const batchSize = 10000
	var trades []*entities.Trade
	var totalProcessed int64
	var bytesRead int64
	var batchCount int

	logger.Info("Starting CSV processing", "file_size_mb", fileSize/(1024*1024))

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV record at line %d: %w", totalProcessed+1, err)
		}

		// Parse CSV record: ID,Price,Quantity,QuoteQuantity,Timestamp,IsBuyerMaker,IsBestMatch
		trade, err := parseCSVRecord(record, symbol)
		if err != nil {
			logger.Warn("Failed to parse record", "line", totalProcessed+1, "error", err)
			continue
		}

		trades = append(trades, trade)
		totalProcessed++

		// Update bytes read (approximate)
		for _, field := range record {
			bytesRead += int64(len(field) + 1) // +1 for delimiter
		}

		// Save batch when full
		if len(trades) >= batchSize {
			if err := repository.SaveBatch(ctx, trades); err != nil {
				return fmt.Errorf("failed to save batch: %w", err)
			}

			batchCount++
			progress := float64(bytesRead) / float64(fileSize) * 100

			logger.Info("Saved batch",
				"batch", batchCount,
				"trades_in_batch", len(trades),
				"total_processed", totalProcessed,
				"progress_percent", fmt.Sprintf("%.2f", progress))

			// Reset batch
			trades = trades[:0]
		}
	}

	// Save remaining trades
	if len(trades) > 0 {
		if err := repository.SaveBatch(ctx, trades); err != nil {
			return fmt.Errorf("failed to save final batch: %w", err)
		}
		batchCount++

		logger.Info("Saved final batch",
			"batch", batchCount,
			"trades_in_batch", len(trades),
			"total_processed", totalProcessed,
			"progress_percent", "100.00")
	}

	logger.Info("CSV import completed",
		"total_trades", totalProcessed,
		"total_batches", batchCount,
		"symbol", symbol)

	return nil
}

func parseCSVRecord(record []string, symbol string) (*entities.Trade, error) {
	if len(record) != 7 {
		return nil, fmt.Errorf("expected 7 fields, got %d", len(record))
	}

	// Parse ID
	id := strings.TrimSpace(record[0])

	// Parse Price
	price, err := strconv.ParseFloat(strings.TrimSpace(record[1]), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse price: %w", err)
	}

	// Parse Quantity
	quantity, err := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse quantity: %w", err)
	}

	// Parse Timestamp (microseconds)
	timestampMicros, err := strconv.ParseInt(strings.TrimSpace(record[4]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	// Convert microseconds to time
	tradeTime := time.Unix(0, timestampMicros*int64(time.Microsecond))

	// Parse IsBuyerMaker
	isBuyerMaker, err := strconv.ParseBool(strings.TrimSpace(record[5]))
	if err != nil {
		return nil, fmt.Errorf("failed to parse is_buyer_maker: %w", err)
	}

	// Create trade entity
	trade := entities.NewTrade(
		id,
		symbol,
		price,
		quantity,
		tradeTime,
		isBuyerMaker,
		tradeTime, // Using trade time as event time for CSV data
	)

	return trade, nil
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
