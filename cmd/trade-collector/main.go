package main

import (
	internalBinance "alarket/internal/binance"
	"alarket/internal/binance/processors"
	"alarket/internal/config"
	"alarket/internal/connector"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/adshao/go-binance/v2"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("Trade Collector has started")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	binance.UseTestnet = cfg.Binance.UseTestnet

	binanceHandler := internalBinance.NewHandler(logger)
	connInstance := connector.New(`wss://stream.binance.com:443/ws`, binanceHandler.Handle, logger)
	connInstance.Run()

	tickerService := processors.NewTickerService(logger)
	tickersToAdd := getTickers(tickerService)

	err = connInstance.SubscribeStreams(tickersToAdd)
	if err != nil {
		logger.Error("Failed to subscribe to streams", "error", err)
		os.Exit(1)
	}

	logger.Info("Successfully subscribed to ticker streams", "ticker_count", len(tickersToAdd))

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		sig := <-sigs
		logger.Info("Received shutdown signal", "signal", sig.String())
		connInstance.ClosePool()
		done <- true
	}()

	<-done
	logger.Info("Trade Collector stopped")

}

func getTickers(tickerService *processors.TickerService) []string {
	tickers, err := tickerService.GetTickers()

	if err != nil {
		os.Exit(1)
	}

	var result []string
	for _, ticker := range tickers {
		result = append(result, ticker.Symbol)
	}

	return result
}
