package clickhouse

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"alarket/internal/domain/entities"
	"alarket/internal/domain/repositories"
)

type TradeBatchProcessor struct {
	tradeRepo    repositories.TradeRepository
	logger       *slog.Logger
	batchSize    int
	flushTimeout time.Duration
	trades       []*entities.Trade
	mu           sync.Mutex
	flushTimer   *time.Timer
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func NewTradeBatchProcessor(
	tradeRepo repositories.TradeRepository,
	logger *slog.Logger,
	batchSize int,
	flushTimeout time.Duration,
) *TradeBatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	processor := &TradeBatchProcessor{
		tradeRepo:    tradeRepo,
		logger:       logger,
		batchSize:    batchSize,
		flushTimeout: flushTimeout,
		trades:       make([]*entities.Trade, 0, batchSize),
		ctx:          ctx,
		cancel:       cancel,
	}

	processor.flushTimer = time.NewTimer(flushTimeout)
	processor.flushTimer.Stop() // Don't start timer until first trade

	// Start background flush routine
	processor.wg.Add(1)
	go processor.flushRoutine()

	return processor
}

func (p *TradeBatchProcessor) AddTrade(trade *entities.Trade) error {
	if err := trade.Validate(); err != nil {
		p.logger.Error("Invalid trade data", "error", err, "tradeID", trade.ID)
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Add trade to batch
	p.trades = append(p.trades, trade)

	// Start timer if this is the first trade in batch
	if len(p.trades) == 1 {
		p.flushTimer.Reset(p.flushTimeout)
	}

	// Check if batch is full
	if len(p.trades) >= p.batchSize {
		p.flushBatch()
	}

	p.logger.Debug("Trade added to batch",
		"tradeID", trade.ID,
		"symbol", trade.Symbol,
		"batchSize", len(p.trades),
	)

	return nil
}

func (p *TradeBatchProcessor) flushRoutine() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			// Flush remaining trades on shutdown
			p.mu.Lock()
			if len(p.trades) > 0 {
				p.flushBatch()
			}
			p.mu.Unlock()
			return

		case <-p.flushTimer.C:
			p.mu.Lock()
			if len(p.trades) > 0 {
				p.flushBatch()
			}
			p.mu.Unlock()
		}
	}
}

func (p *TradeBatchProcessor) flushBatch() {
	if len(p.trades) == 0 {
		return
	}

	// Create a copy of trades to flush
	batch := make([]*entities.Trade, len(p.trades))
	copy(batch, p.trades)

	// Clear the current batch
	p.trades = p.trades[:0]
	p.flushTimer.Stop()

	// Flush to database (release lock first to avoid blocking new trades)
	go func(trades []*entities.Trade) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := p.tradeRepo.SaveBatch(ctx, trades); err != nil {
			p.logger.Error("Failed to flush trade batch",
				"error", err,
				"batchSize", len(trades),
			)
			// TODO: Consider implementing retry logic or dead letter queue
		} else {
			p.logger.Info("Trade batch flushed successfully",
				"batchSize", len(trades),
			)
		}
	}(batch)
}

func (p *TradeBatchProcessor) Close() error {
	p.cancel()
	p.wg.Wait()

	// Ensure timer is stopped
	if p.flushTimer != nil {
		p.flushTimer.Stop()
	}

	return nil
}
