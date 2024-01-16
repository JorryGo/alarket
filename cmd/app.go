package main

import (
	internalBinance "alarket/internal/binance"
	"alarket/internal/connector"
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.DateTime})

	log.Info().Msg(`Scrapper has started`)

	connInstance := connector.New(`wss://stream.binance.com:443/ws`, internalBinance.NewRequestHandler().Handle)
	connInstance.Run()

	binanceClient := binance.NewClient(``, ``)
	tickerList, err := binanceClient.NewListBookTickersService().Do(context.Background())

	if err != nil {
		log.Fatal().Err(err)
	}

	tickersToAdd := make([]string, 0, len(tickerList))
	for _, ticker := range tickerList {
		tickersToAdd = append(tickersToAdd, ticker.Symbol)
	}

	err = connInstance.SubscribeStreams(tickersToAdd)

	if err != nil {
		log.Warn().Err(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill)
	done := make(chan bool, 1)

	go func() {
		sig := <-sigs
		fmt.Println(sig)
		connInstance.ClosePool()
		done <- true
	}()

	<-done
	fmt.Println(`exiting`)

}
