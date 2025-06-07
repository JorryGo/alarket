package binance

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/adshao/go-binance/v2"
	"alarket/internal/domain/entities"
)

type SymbolFetcher struct {
	client     *binance.Client
	logger     *slog.Logger
	useTestnet bool
}

func NewSymbolFetcher(apiKey, secretKey string, useTestnet bool, logger *slog.Logger) *SymbolFetcher {
	client := binance.NewClient(apiKey, secretKey)
	
	if useTestnet {
		binance.UseTestnet = true
	}

	return &SymbolFetcher{
		client:     client,
		logger:     logger,
		useTestnet: useTestnet,
	}
}

func (f *SymbolFetcher) FetchAllSymbols(ctx context.Context) ([]*entities.Symbol, error) {
	exchangeInfo, err := f.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exchange info: %w", err)
	}

	symbols := make([]*entities.Symbol, 0, len(exchangeInfo.Symbols))
	for _, s := range exchangeInfo.Symbols {
		status := entities.SymbolStatus(s.Status)
		symbol := entities.NewSymbol(s.Symbol, s.BaseAsset, s.QuoteAsset, status)
		
		// Set trading flags based on permissions
		for _, perm := range s.Permissions {
			switch perm {
			case "SPOT":
				symbol.IsSpotTrading = true
			case "MARGIN":
				symbol.IsMarginTrading = true
			}
		}

		symbols = append(symbols, symbol)
	}

	f.logger.Info("Fetched symbols from exchange", "count", len(symbols))
	return symbols, nil
}