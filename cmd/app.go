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

	connInstance := connector.New(`wss://stream.binance.com:443/ws`, internalBinance.Handle)
	connInstance.Run()

	tickersToAdd := getTickers()

	err := connInstance.SubscribeStreams(tickersToAdd)

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

func getTickers() []string {
	binanceClient := binance.NewClient(``, ``)
	tickerList, err := binanceClient.NewListBookTickersService().Do(context.Background())

	if err != nil {
		log.Fatal().Err(err)
	}

	usdtTickers := make([]*binance.BookTicker, 0)
	for _, ticker := range tickerList {
		if ticker.Symbol[len(ticker.Symbol)-4:] == `USDT` {
			usdtTickers = append(usdtTickers, ticker)
		}
		if ticker.Symbol[:4] == `USDT` {
			usdtTickers = append(usdtTickers, ticker)
		}
	}

	tickersToAdd := make([]string, 0, len(usdtTickers))
	for _, ticker := range usdtTickers {
		tickersToAdd = append(tickersToAdd, ticker.Symbol)
	}

	return tickersToAdd
}
