package container

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	
	appservices "alarket/internal/application/services"
	"alarket/internal/application/usecases"
	"alarket/internal/domain/repositories"
	domainservices "alarket/internal/domain/services"
	"alarket/internal/infrastructure/binance"
	"alarket/internal/infrastructure/clickhouse"
	"alarket/internal/infrastructure/config"
)

type Container struct {
	Config *config.Config
	Logger *slog.Logger
	
	// Repositories
	TradeRepository      repositories.TradeRepository
	BookTickerRepository repositories.BookTickerRepository
	SymbolRepository     repositories.SymbolRepository
	
	// Batch Processors
	TradeBatchProcessor      *clickhouse.TradeBatchProcessor
	BookTickerBatchProcessor *clickhouse.BookTickerBatchProcessor
	
	// Use Cases
	ProcessTradeUseCase      *usecases.ProcessTradeEventUseCase
	ProcessBookTickerUseCase *usecases.ProcessBookTickerEventUseCase
	SubscribeToSymbolsUseCase *usecases.SubscribeToSymbolsUseCase
	
	// Services
	ExchangeClient domainservices.ExchangeClient
	EventHandler   *appservices.EventHandler
	
	// Infrastructure
	DB *sql.DB
}

func New(ctx context.Context) (*Container, error) {
	c := &Container{}
	
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	c.Config = cfg
	
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
	
	c.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	
	// Setup database
	if err := c.setupDatabase(ctx); err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}
	
	// Setup repositories
	if err := c.setupRepositories(ctx); err != nil {
		return nil, fmt.Errorf("failed to setup repositories: %w", err)
	}
	
	// Setup batch processors
	c.setupBatchProcessors()
	
	// Setup use cases
	c.setupUseCases()
	
	// Setup services
	if err := c.setupServices(); err != nil {
		return nil, fmt.Errorf("failed to setup services: %w", err)
	}
	
	return c, nil
}

func (c *Container) setupDatabase(ctx context.Context) error {
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s?debug=%t",
		c.Config.ClickHouse.Username,
		c.Config.ClickHouse.Password,
		c.Config.ClickHouse.Host,
		c.Config.ClickHouse.Port,
		c.Config.ClickHouse.Database,
		c.Config.ClickHouse.Debug,
	)
	
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	
	c.DB = db
	
	// Run migrations
	migrator := clickhouse.NewMigrator(db, c.Logger)
	if err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	
	return nil
}

func (c *Container) setupRepositories(ctx context.Context) error {
	// Setup repositories
	c.TradeRepository = clickhouse.NewTradeRepository(c.DB)
	c.BookTickerRepository = clickhouse.NewBookTickerRepository(c.DB)
	
	// Fetch symbols from Binance
	symbolFetcher := binance.NewSymbolFetcher(
		c.Config.Binance.APIKey,
		c.Config.Binance.SecretKey,
		c.Config.Binance.UseTestnet,
		c.Logger,
	)
	
	symbols, err := symbolFetcher.FetchAllSymbols(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch symbols: %w", err)
	}
	
	c.SymbolRepository = clickhouse.NewSymbolRepository(c.DB, symbols)
	
	return nil
}

func (c *Container) setupBatchProcessors() {
	// Create batch processors with configurable settings
	flushTimeout := time.Duration(c.Config.App.BatchFlushTimeoutMs) * time.Millisecond
	
	c.TradeBatchProcessor = clickhouse.NewTradeBatchProcessor(
		c.TradeRepository,
		c.Logger,
		c.Config.App.BatchSize,
		flushTimeout,
	)
	
	c.BookTickerBatchProcessor = clickhouse.NewBookTickerBatchProcessor(
		c.BookTickerRepository,
		c.Logger,
		c.Config.App.BatchSize,
		flushTimeout,
	)
}

func (c *Container) setupUseCases() {
	c.ProcessTradeUseCase = usecases.NewProcessTradeEventUseCase(
		c.TradeBatchProcessor,
		c.Logger,
	)
	
	c.ProcessBookTickerUseCase = usecases.NewProcessBookTickerEventUseCase(
		c.BookTickerBatchProcessor,
		c.Logger,
	)
	
	c.SubscribeToSymbolsUseCase = usecases.NewSubscribeToSymbolsUseCase(
		c.SymbolRepository,
		c.ExchangeClient,
		c.Logger,
	)
}

func (c *Container) setupServices() error {
	// Create event handler first
	c.EventHandler = appservices.NewEventHandler(
		c.ProcessTradeUseCase,
		c.ProcessBookTickerUseCase,
		c.Logger,
	)
	
	// Create exchange client with event handler
	c.ExchangeClient = binance.NewClient(
		c.Logger,
		c.Config.Binance.UseTestnet,
		func(message []byte) error {
			return c.EventHandler.HandleMessage(context.Background(), message)
		},
	)
	
	// Update use case with exchange client
	c.SubscribeToSymbolsUseCase = usecases.NewSubscribeToSymbolsUseCase(
		c.SymbolRepository,
		c.ExchangeClient,
		c.Logger,
	)
	
	return nil
}

func (c *Container) Close() error {
	// Close batch processors first to flush remaining data
	if c.TradeBatchProcessor != nil {
		if err := c.TradeBatchProcessor.Close(); err != nil {
			c.Logger.Error("Failed to close trade batch processor", "error", err)
		}
	}
	
	if c.BookTickerBatchProcessor != nil {
		if err := c.BookTickerBatchProcessor.Close(); err != nil {
			c.Logger.Error("Failed to close book ticker batch processor", "error", err)
		}
	}
	
	if c.ExchangeClient != nil {
		if err := c.ExchangeClient.Close(); err != nil {
			c.Logger.Error("Failed to close exchange client", "error", err)
		}
	}
	
	if c.DB != nil {
		if err := c.DB.Close(); err != nil {
			c.Logger.Error("Failed to close database", "error", err)
		}
	}
	
	return nil
}