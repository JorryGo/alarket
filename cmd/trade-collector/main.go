package main

import (
	internalBinance "alarket/internal/binance"
	"alarket/internal/binance/processors"
	"alarket/internal/config"
	"alarket/internal/connector"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.DateTime})

	log.Info().Msg(`Trade Collector has started`)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	binance.UseTestnet = cfg.Binance.UseTestnet

	connInstance := connector.New(`wss://stream.binance.com:443/ws`, internalBinance.Handle)
	connInstance.Run()

	tickersToAdd := getTickers()

	err = connInstance.SubscribeStreams(tickersToAdd)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to subscribe to streams")
	}

	log.Info().Int("ticker_count", len(tickersToAdd)).Msg("Successfully subscribed to ticker streams")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		sig := <-sigs
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		connInstance.ClosePool()
		done <- true
	}()

	<-done
	log.Info().Msg("Trade Collector stopped")

}

func getTickers() []string {
	tickers, err := processors.GetTickers()

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get tickers")
	}

	var result []string
	for _, ticker := range tickers {
		result = append(result, ticker.Symbol)
	}

	return result
}
