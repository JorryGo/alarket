package main

import (
	"alarket/internal/infrastructure/container"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := container.New(ctx)
	if err != nil {
		slog.Error("Failed to initialize container", "error", err)
		os.Exit(1)
	}
	logger := c.Logger

	defer func() {
		if err := c.Close(); err != nil {
			logger.Error("Failed to close container", "error", err)
		}
	}()
	logger.Info("Trade Collector started",
		"testnet", c.Config.Binance.UseTestnet,
		"subscribeTrades", c.Config.App.SubscribeTrades,
		"subscribeBookTickers", c.Config.App.SubscribeBookTickers,
	)

	// Subscribe to symbols
	if err := c.SubscribeToSymbolsUseCase.Execute(
		ctx,
		c.Config.App.SubscribeTrades,
		c.Config.App.SubscribeBookTickers,
	); err != nil {
		logger.Error("Failed to subscribe to symbols", "error", err)
		os.Exit(1)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutdown signal received")

	// Cancel context to stop all operations
	cancel()

	// Close container resources
	if err := c.Close(); err != nil {
		logger.Error("Error during shutdown", "error", err)
	}

	logger.Info("Trade Collector stopped")
}
