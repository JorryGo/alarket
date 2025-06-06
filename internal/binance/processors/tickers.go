package processors

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/adshao/go-binance/v2"
)

type SymbolTree struct {
	SymbolName  string
	Symbol      *binance.Symbol
	LotMinQty   float64
	LotStepSize float64
	From        *SymbolTree
	To          *map[string]*SymbolTree
}

type TickerService struct {
	logger *slog.Logger
	client *binance.Client
}

func NewTickerService(logger *slog.Logger) *TickerService {
	return &TickerService{
		logger: logger,
		client: binance.NewClient(``, ``),
	}
}

func (s *TickerService) GetTickers() ([]*binance.Symbol, error) {
	tickerList, err := s.client.NewExchangeInfoService().Do(context.Background())

	if err != nil || tickerList == nil {
		s.logger.Error("Failed to get exchange info", "error", err)
		return nil, err
	}

	res := make([]*binance.Symbol, 0, len(tickerList.Symbols))

	for _, ticker := range tickerList.Symbols {
		internalTicker := ticker
		if ticker.Status == `TRADING` {
			res = append(res, &internalTicker)
		}
	}

	return res, nil
}

func (s *TickerService) GetTickersForMap() (*map[string]*SymbolTree, error) {
	tickers, err := s.GetTickers()
	if err != nil {
		return nil, err
	}

	return findLoops(tickers, nil, []string{}, 0), nil
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

		minLotQty, _ := strconv.ParseFloat(symbol.LotSizeFilter().MinQuantity, 64)
		lotStepSize, _ := strconv.ParseFloat(symbol.LotSizeFilter().StepSize, 64)

		st := &SymbolTree{
			SymbolName:  symbol.Symbol,
			Symbol:      symbol,
			LotMinQty:   minLotQty,
			LotStepSize: lotStepSize,
			From:        from,
		}

		st.To = findLoops(symbols, st, []string{symbol.QuoteAsset, symbol.BaseAsset}, deep)

		if len(*st.To) == 0 && deep != 3 {
			continue
		}

		symbolsTree[symbol.Symbol] = st
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
