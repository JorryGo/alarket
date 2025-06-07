package clickhouse

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"alarket/internal/domain/entities"
	"alarket/internal/domain/repositories"
)

type BookTickerBatchProcessor struct {
	bookTickerRepo repositories.BookTickerRepository
	logger         *slog.Logger
	batchSize      int
	flushTimeout   time.Duration
	bookTickers    []*entities.BookTicker
	mu             sync.Mutex
	flushTimer     *time.Timer
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

func NewBookTickerBatchProcessor(
	bookTickerRepo repositories.BookTickerRepository,
	logger *slog.Logger,
	batchSize int,
	flushTimeout time.Duration,
) *BookTickerBatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	processor := &BookTickerBatchProcessor{
		bookTickerRepo: bookTickerRepo,
		logger:         logger,
		batchSize:      batchSize,
		flushTimeout:   flushTimeout,
		bookTickers:    make([]*entities.BookTicker, 0, batchSize),
		ctx:            ctx,
		cancel:         cancel,
	}

	processor.flushTimer = time.NewTimer(flushTimeout)
	processor.flushTimer.Stop() // Don't start timer until first book ticker

	// Start background flush routine
	processor.wg.Add(1)
	go processor.flushRoutine()

	return processor
}

func (p *BookTickerBatchProcessor) AddBookTicker(ticker *entities.BookTicker) error {
	if err := ticker.Validate(); err != nil {
		p.logger.Error("Invalid book ticker data", "error", err, "symbol", ticker.Symbol)
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Add book ticker to batch
	p.bookTickers = append(p.bookTickers, ticker)

	// Start timer if this is the first book ticker in batch
	if len(p.bookTickers) == 1 {
		p.flushTimer.Reset(p.flushTimeout)
	}

	// Check if batch is full
	if len(p.bookTickers) >= p.batchSize {
		p.flushBatch()
	}

	p.logger.Debug("Book ticker added to batch",
		"symbol", ticker.Symbol,
		"bidPrice", ticker.BestBidPrice,
		"askPrice", ticker.BestAskPrice,
		"batchSize", len(p.bookTickers),
	)

	return nil
}

func (p *BookTickerBatchProcessor) flushRoutine() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			// Flush remaining book tickers on shutdown
			p.mu.Lock()
			if len(p.bookTickers) > 0 {
				p.flushBatch()
			}
			p.mu.Unlock()
			return

		case <-p.flushTimer.C:
			p.mu.Lock()
			if len(p.bookTickers) > 0 {
				p.flushBatch()
			}
			p.mu.Unlock()
		}
	}
}

func (p *BookTickerBatchProcessor) flushBatch() {
	if len(p.bookTickers) == 0 {
		return
	}

	// Create a copy of book tickers to flush
	batch := make([]*entities.BookTicker, len(p.bookTickers))
	copy(batch, p.bookTickers)

	// Clear the current batch
	p.bookTickers = p.bookTickers[:0]
	p.flushTimer.Stop()

	// Flush to database (release lock first to avoid blocking new book tickers)
	go func(tickers []*entities.BookTicker) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := p.bookTickerRepo.SaveBatch(ctx, tickers); err != nil {
			p.logger.Error("Failed to flush book ticker batch",
				"error", err,
				"batchSize", len(tickers),
			)
			// TODO: Consider implementing retry logic or dead letter queue
		} else {
			p.logger.Info("Book ticker batch flushed successfully",
				"batchSize", len(tickers),
			)
		}
	}(batch)
}

func (p *BookTickerBatchProcessor) Close() error {
	p.cancel()
	p.wg.Wait()

	// Ensure timer is stopped
	if p.flushTimer != nil {
		p.flushTimer.Stop()
	}

	return nil
}
