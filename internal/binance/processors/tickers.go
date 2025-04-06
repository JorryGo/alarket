package processors

import (
	"context"
	"github.com/adshao/go-binance/v2"
	"github.com/rs/zerolog/log"
)

type SymbolTree struct {
	SymbolName string
	Symbol     *binance.Symbol
	From       *SymbolTree
	To         *map[string]*SymbolTree
}

func GetTickersForMap() (*map[string]*SymbolTree, error) {
	binanceClient := binance.NewClient(``, ``)
	tickerList, err := binanceClient.NewExchangeInfoService().Do(context.Background())

	if err != nil || tickerList == nil {
		log.Fatal().Err(err)
		return nil, err
	}

	res := make([]*binance.Symbol, 0, len(tickerList.Symbols))

	for _, ticker := range tickerList.Symbols {
		internalTicker := ticker
		isUsdcPair := ticker.BaseAsset == "USDC" || ticker.QuoteAsset == "USDC"
		if ticker.Status == `TRADING` && !isUsdcPair {
			res = append(res, &internalTicker)
		}
	}

	return findLoops(res, nil, []string{}, 0), nil
}

func findLoops(symbols []*binance.Symbol, from *SymbolTree, symbolToFind []string, deep int) *map[string]*SymbolTree {
	symbolsTree := make(map[string]*SymbolTree)
	deep++

	if deep >= 4 {
		return &symbolsTree
	}

	for _, symbol := range symbols {
		isSearched := inArray(symbolToFind, symbol.BaseAsset) || inArray(symbolToFind, symbol.QuoteAsset)
		if !isSearched && len(symbolToFind) > 0 {
			continue
		}

		if isHasBeenProcessed(symbol.Symbol, from) {
			continue
		}

		// Для того чтобы получать цепочти начинающиеся и заканчивающиеся на usdt
		isUsdtPair := symbol.BaseAsset == "USDT" || symbol.QuoteAsset == "USDT"
		if (deep == 1 || deep == 3) && !isUsdtPair {
			continue
		}
		if deep == 2 && isUsdtPair {
			continue
		}

		st := &SymbolTree{
			SymbolName: symbol.Symbol,
			Symbol:     symbol,
			From:       from,
		}

		symbolsTree[symbol.Symbol] = st

		st.To = findLoops(symbols, st, []string{symbol.QuoteAsset, symbol.BaseAsset}, deep)
	}

	return &symbolsTree
}

func inArray(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func isHasBeenProcessed(symbol string, tree *SymbolTree) bool {
	if tree == nil {
		return false
	}

	if symbol == tree.SymbolName {
		return true
	}

	return isHasBeenProcessed(symbol, tree.From)
}
