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
	slog.SetDefault(logger)

	slog.Info("Trade Collector has started")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	binance.UseTestnet = cfg.Binance.UseTestnet

	connInstance := connector.New(`wss://stream.binance.com:443/ws`, internalBinance.Handle)
	connInstance.Run()

	tickersToAdd := getTickers()

	err = connInstance.SubscribeStreams(tickersToAdd)
	if err != nil {
		slog.Error("Failed to subscribe to streams", "error", err)
		os.Exit(1)
	}

	slog.Info("Successfully subscribed to ticker streams", "ticker_count", len(tickersToAdd))

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		sig := <-sigs
		slog.Info("Received shutdown signal", "signal", sig.String())
		connInstance.ClosePool()
		done <- true
	}()

	<-done
	slog.Info("Trade Collector stopped")

}

func getTickers() []string {
	tickers, err := processors.GetTickers()

	if err != nil {
		slog.Error("Failed to get tickers", "error", err)
		os.Exit(1)
	}

	var result []string
	for _, ticker := range tickers {
		result = append(result, ticker.Symbol)
	}

	return result
}
