package usecases

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"alarket/internal/domain/entities"
	"alarket/internal/domain/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewSubscribeToSymbolsUseCase(t *testing.T) {
	mockSymbolRepo := new(mocks.MockSymbolRepository)
	mockExchangeClient := new(mocks.MockExchangeClient)
	logger := slog.Default()
	
	uc := NewSubscribeToSymbolsUseCase(mockSymbolRepo, mockExchangeClient, logger)
	
	assert.NotNil(t, uc)
	assert.NotNil(t, uc.symbolRepo)
	assert.NotNil(t, uc.exchangeClient)
	assert.NotNil(t, uc.logger)
}

func TestSubscribeToSymbolsUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	
	t.Run("successful subscription to trades only", func(t *testing.T) {
		mockSymbolRepo := new(mocks.MockSymbolRepository)
		mockExchangeClient := new(mocks.MockExchangeClient)
		
		// Setup active symbols
		activeSymbols := []*entities.Symbol{
			{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
				Status:     entities.SymbolStatusTrading,
			},
			{
				Name:       "ETHUSDT",
				BaseAsset:  "ETH",
				QuoteAsset: "USDT",
				Status:     entities.SymbolStatusTrading,
			},
		}
		
		mockSymbolRepo.On("GetActiveUsdt", ctx).Return(activeSymbols, nil)
		mockExchangeClient.On("SubscribeToTrades", ctx, []string{"BTCUSDT", "ETHUSDT"}).Return(nil)
		
		uc := NewSubscribeToSymbolsUseCase(mockSymbolRepo, mockExchangeClient, logger)
		
		err := uc.Execute(ctx, true, false)
		assert.NoError(t, err)
		
		mockSymbolRepo.AssertExpectations(t)
		mockExchangeClient.AssertExpectations(t)
		mockExchangeClient.AssertNotCalled(t, "SubscribeToBookTickers", mock.Anything, mock.Anything)
	})
	
	t.Run("successful subscription to book tickers only", func(t *testing.T) {
		mockSymbolRepo := new(mocks.MockSymbolRepository)
		mockExchangeClient := new(mocks.MockExchangeClient)
		
		activeSymbols := []*entities.Symbol{
			{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
				Status:     entities.SymbolStatusTrading,
			},
		}
		
		mockSymbolRepo.On("GetActiveUsdt", ctx).Return(activeSymbols, nil)
		mockExchangeClient.On("SubscribeToBookTickers", ctx, []string{"BTCUSDT"}).Return(nil)
		
		uc := NewSubscribeToSymbolsUseCase(mockSymbolRepo, mockExchangeClient, logger)
		
		err := uc.Execute(ctx, false, true)
		assert.NoError(t, err)
		
		mockSymbolRepo.AssertExpectations(t)
		mockExchangeClient.AssertExpectations(t)
		mockExchangeClient.AssertNotCalled(t, "SubscribeToTrades", mock.Anything, mock.Anything)
	})
	
	t.Run("successful subscription to both trades and book tickers", func(t *testing.T) {
		mockSymbolRepo := new(mocks.MockSymbolRepository)
		mockExchangeClient := new(mocks.MockExchangeClient)
		
		activeSymbols := []*entities.Symbol{
			{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
				Status:     entities.SymbolStatusTrading,
			},
			{
				Name:       "ETHUSDT",
				BaseAsset:  "ETH",
				QuoteAsset: "USDT",
				Status:     entities.SymbolStatusTrading,
			},
			{
				Name:       "BNBUSDT",
				BaseAsset:  "BNB",
				QuoteAsset: "USDT",
				Status:     entities.SymbolStatusTrading,
			},
		}
		
		expectedSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
		
		mockSymbolRepo.On("GetActiveUsdt", ctx).Return(activeSymbols, nil)
		mockExchangeClient.On("SubscribeToTrades", ctx, expectedSymbols).Return(nil)
		mockExchangeClient.On("SubscribeToBookTickers", ctx, expectedSymbols).Return(nil)
		
		uc := NewSubscribeToSymbolsUseCase(mockSymbolRepo, mockExchangeClient, logger)
		
		err := uc.Execute(ctx, true, true)
		assert.NoError(t, err)
		
		mockSymbolRepo.AssertExpectations(t)
		mockExchangeClient.AssertExpectations(t)
	})
	
	t.Run("no subscription when both flags are false", func(t *testing.T) {
		mockSymbolRepo := new(mocks.MockSymbolRepository)
		mockExchangeClient := new(mocks.MockExchangeClient)
		
		activeSymbols := []*entities.Symbol{
			{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
				Status:     entities.SymbolStatusTrading,
			},
		}
		
		mockSymbolRepo.On("GetActiveUsdt", ctx).Return(activeSymbols, nil)
		
		uc := NewSubscribeToSymbolsUseCase(mockSymbolRepo, mockExchangeClient, logger)
		
		err := uc.Execute(ctx, false, false)
		assert.NoError(t, err)
		
		mockSymbolRepo.AssertExpectations(t)
		mockExchangeClient.AssertNotCalled(t, "SubscribeToTrades", mock.Anything, mock.Anything)
		mockExchangeClient.AssertNotCalled(t, "SubscribeToBookTickers", mock.Anything, mock.Anything)
	})
	
	t.Run("error getting active symbols", func(t *testing.T) {
		mockSymbolRepo := new(mocks.MockSymbolRepository)
		mockExchangeClient := new(mocks.MockExchangeClient)
		
		expectedErr := errors.New("database error")
		mockSymbolRepo.On("GetActiveUsdt", ctx).Return(nil, expectedErr)
		
		uc := NewSubscribeToSymbolsUseCase(mockSymbolRepo, mockExchangeClient, logger)
		
		err := uc.Execute(ctx, true, true)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		
		mockSymbolRepo.AssertExpectations(t)
		mockExchangeClient.AssertNotCalled(t, "SubscribeToTrades", mock.Anything, mock.Anything)
		mockExchangeClient.AssertNotCalled(t, "SubscribeToBookTickers", mock.Anything, mock.Anything)
	})
	
	t.Run("error subscribing to trades", func(t *testing.T) {
		mockSymbolRepo := new(mocks.MockSymbolRepository)
		mockExchangeClient := new(mocks.MockExchangeClient)
		
		activeSymbols := []*entities.Symbol{
			{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
				Status:     entities.SymbolStatusTrading,
			},
		}
		
		expectedErr := errors.New("subscription error")
		mockSymbolRepo.On("GetActiveUsdt", ctx).Return(activeSymbols, nil)
		mockExchangeClient.On("SubscribeToTrades", ctx, []string{"BTCUSDT"}).Return(expectedErr)
		
		uc := NewSubscribeToSymbolsUseCase(mockSymbolRepo, mockExchangeClient, logger)
		
		err := uc.Execute(ctx, true, false)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		
		mockSymbolRepo.AssertExpectations(t)
		mockExchangeClient.AssertExpectations(t)
	})
	
	t.Run("error subscribing to book tickers", func(t *testing.T) {
		mockSymbolRepo := new(mocks.MockSymbolRepository)
		mockExchangeClient := new(mocks.MockExchangeClient)
		
		activeSymbols := []*entities.Symbol{
			{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
				Status:     entities.SymbolStatusTrading,
			},
		}
		
		expectedErr := errors.New("book ticker subscription error")
		mockSymbolRepo.On("GetActiveUsdt", ctx).Return(activeSymbols, nil)
		mockExchangeClient.On("SubscribeToBookTickers", ctx, []string{"BTCUSDT"}).Return(expectedErr)
		
		uc := NewSubscribeToSymbolsUseCase(mockSymbolRepo, mockExchangeClient, logger)
		
		err := uc.Execute(ctx, false, true)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		
		mockSymbolRepo.AssertExpectations(t)
		mockExchangeClient.AssertExpectations(t)
	})
	
	t.Run("empty active symbols list", func(t *testing.T) {
		mockSymbolRepo := new(mocks.MockSymbolRepository)
		mockExchangeClient := new(mocks.MockExchangeClient)
		
		mockSymbolRepo.On("GetActiveUsdt", ctx).Return([]*entities.Symbol{}, nil)
		mockExchangeClient.On("SubscribeToTrades", ctx, []string{}).Return(nil)
		mockExchangeClient.On("SubscribeToBookTickers", ctx, []string{}).Return(nil)
		
		uc := NewSubscribeToSymbolsUseCase(mockSymbolRepo, mockExchangeClient, logger)
		
		err := uc.Execute(ctx, true, true)
		assert.NoError(t, err)
		
		mockSymbolRepo.AssertExpectations(t)
		mockExchangeClient.AssertExpectations(t)
	})
}