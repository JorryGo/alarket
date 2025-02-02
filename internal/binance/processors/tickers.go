package processors

import (
	"context"
	"fmt"
	"github.com/adshao/go-binance/v2"
	"github.com/rs/zerolog/log"
)

type SymbolTree struct {
	From string
	To   map[string]SymbolTree
}

func GetTickersForMap() {
	binanceClient := binance.NewClient(``, ``)
	tickerList, err := binanceClient.NewExchangeInfoService().Symbols("BTCUSDT", "ETHBTC", "ETHUSDT").Do(context.Background())

	if err != nil || tickerList == nil {
		log.Fatal().Err(err)
		return
	}

	res := make([]binance.Symbol, 0, len(tickerList.Symbols))

	for _, ticker := range tickerList.Symbols {
		if ticker.Status == `TRADING` {
			res = append(res, ticker)
		}
	}

	tree := findLoops(res, []string{}, 0)

	fmt.Println(tickerList, err, tree)

}

func findLoops(symbols []binance.Symbol, symbolToFind []string, deep int) map[string]SymbolTree {
	symbolsTree := make(map[string]SymbolTree)
	deep++

	if deep > 4 {
		return symbolsTree
	}

	for _, symbol := range symbols {
		if !inArray(symbolToFind, symbol.BaseAsset) && len(symbolToFind) > 0 {
			continue
		}

		to := findLoops(symbols, []string{symbol.QuoteAsset, symbol.BaseAsset}, deep)

		symbolsTree[symbol.BaseAsset] = SymbolTree{
			From: symbol.BaseAsset,
			To:   to,
		}
	}

	return symbolsTree
}

func inArray(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
