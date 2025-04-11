package main

import (
	internalBinance "alarket/internal/binance"
	"alarket/internal/binance/processors"
	"alarket/internal/connector"
	trader2 "alarket/internal/trader"
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

	tree, err := processors.GetTickersForMap()
	if err != nil {
		log.Fatal().Err(err).Msg("Error getting tickers")
	}

	tickersToAddMap := make(map[string]struct{}, len(*tree))
	getTickersFromTree(tree, &tickersToAddMap)
	tickersToAdd := make([]string, 0, len(tickersToAddMap))
	for key := range tickersToAddMap {
		tickersToAdd = append(tickersToAdd, key)
	}

	trader := trader2.InitTrader(tree)

	connInstance := connector.New(`wss://stream.binance.com:443/ws`, internalBinance.Handle, trader)
	connInstance.Run()

	err = connInstance.SubscribeStreams(tickersToAdd)

	if err != nil {
		log.Warn().Err(err)
	}

	ticker := time.NewTicker(time.Second / 100)
	for range ticker.C {
		trader.CheckLoopDiffs("BTCUSDT")
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

func getTickersFromTree(tree *map[string]*processors.SymbolTree, tickersMap *map[string]struct{}) {
	for _, ticker := range *tree {
		(*tickersMap)[ticker.SymbolName] = struct{}{}
		if ticker.To == nil {
			continue
		}

		getTickersFromTree(ticker.To, tickersMap)
	}
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
