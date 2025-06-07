package usecases

import (
	"context"
	"log/slog"

	"alarket/internal/domain/repositories"
	"alarket/internal/domain/services"
)

type SubscribeToSymbolsUseCase struct {
	symbolRepo     repositories.SymbolRepository
	exchangeClient services.ExchangeClient
	logger         *slog.Logger
}

func NewSubscribeToSymbolsUseCase(
	symbolRepo repositories.SymbolRepository,
	exchangeClient services.ExchangeClient,
	logger *slog.Logger,
) *SubscribeToSymbolsUseCase {
	return &SubscribeToSymbolsUseCase{
		symbolRepo:     symbolRepo,
		exchangeClient: exchangeClient,
		logger:         logger,
	}
}

func (uc *SubscribeToSymbolsUseCase) Execute(ctx context.Context, subscribeTrades, subscribeBookTickers bool) error {
	activeSymbols, err := uc.symbolRepo.GetActiveUsdt(ctx)
	if err != nil {
		uc.logger.Error("Failed to get active symbols", "error", err)
		return err
	}

	symbolNames := make([]string, 0, len(activeSymbols))
	for _, symbol := range activeSymbols {
		symbolNames = append(symbolNames, symbol.Name)
	}

	uc.logger.Info("Subscribing to symbols", "count", len(symbolNames))

	if subscribeTrades {
		//if err := uc.exchangeClient.SubscribeToTrades(ctx, symbolNames); err != nil {
		//	uc.logger.Error("Failed to subscribe to trades", "error", err)
		//	return err
		//}
	}

	if subscribeBookTickers {
		if err := uc.exchangeClient.SubscribeToBookTickers(ctx, symbolNames); err != nil {
			uc.logger.Error("Failed to subscribe to book tickers", "error", err)
			return err
		}
	}

	return nil
}
