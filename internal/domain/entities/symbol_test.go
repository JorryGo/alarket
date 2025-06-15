package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSymbol(t *testing.T) {
	symbol := NewSymbol("BTCUSDT", "BTC", "USDT", SymbolStatusTrading)

	assert.NotNil(t, symbol)
	assert.Equal(t, "BTCUSDT", symbol.Name)
	assert.Equal(t, "BTC", symbol.BaseAsset)
	assert.Equal(t, "USDT", symbol.QuoteAsset)
	assert.Equal(t, SymbolStatusTrading, symbol.Status)
	assert.False(t, symbol.IsSpotTrading)
	assert.False(t, symbol.IsMarginTrading)
}

func TestSymbol_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   SymbolStatus
		isActive bool
	}{
		{
			name:     "trading status is active",
			status:   SymbolStatusTrading,
			isActive: true,
		},
		{
			name:     "halt status is not active",
			status:   SymbolStatusHalt,
			isActive: false,
		},
		{
			name:     "break status is not active",
			status:   SymbolStatusBreak,
			isActive: false,
		},
		{
			name:     "auction match status is not active",
			status:   SymbolStatusAuctionMatch,
			isActive: false,
		},
		{
			name:     "end of day status is not active",
			status:   SymbolStatusEndOfDay,
			isActive: false,
		},
		{
			name:     "pre trading status is not active",
			status:   SymbolStatusPreTrading,
			isActive: false,
		},
		{
			name:     "post trading status is not active",
			status:   SymbolStatusPostTrading,
			isActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbol := &Symbol{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
				Status:     tt.status,
			}
			assert.Equal(t, tt.isActive, symbol.IsActive())
		})
	}
}

func TestSymbol_Validate(t *testing.T) {
	tests := []struct {
		name    string
		symbol  *Symbol
		wantErr error
	}{
		{
			name: "valid symbol",
			symbol: &Symbol{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
				Status:     SymbolStatusTrading,
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			symbol: &Symbol{
				Name:       "",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
				Status:     SymbolStatusTrading,
			},
			wantErr: ErrInvalidSymbol,
		},
		{
			name: "empty base asset",
			symbol: &Symbol{
				Name:       "BTCUSDT",
				BaseAsset:  "",
				QuoteAsset: "USDT",
				Status:     SymbolStatusTrading,
			},
			wantErr: ErrInvalidAsset,
		},
		{
			name: "empty quote asset",
			symbol: &Symbol{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "",
				Status:     SymbolStatusTrading,
			},
			wantErr: ErrInvalidAsset,
		},
		{
			name: "both assets empty",
			symbol: &Symbol{
				Name:       "BTCUSDT",
				BaseAsset:  "",
				QuoteAsset: "",
				Status:     SymbolStatusTrading,
			},
			wantErr: ErrInvalidAsset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.symbol.Validate()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSymbol_TradingTypes(t *testing.T) {
	t.Run("spot and margin trading flags", func(t *testing.T) {
		symbol := &Symbol{
			Name:            "BTCUSDT",
			BaseAsset:       "BTC",
			QuoteAsset:      "USDT",
			Status:          SymbolStatusTrading,
			IsSpotTrading:   true,
			IsMarginTrading: true,
		}

		assert.True(t, symbol.IsSpotTrading)
		assert.True(t, symbol.IsMarginTrading)
		assert.NoError(t, symbol.Validate())
	})

	t.Run("status doesn't affect validation", func(t *testing.T) {
		symbol := &Symbol{
			Name:       "BTCUSDT",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
			Status:     SymbolStatus("UNKNOWN_STATUS"), // Invalid status
		}

		// Validation doesn't check status validity
		assert.NoError(t, symbol.Validate())
		assert.False(t, symbol.IsActive())
	})
}

func TestSymbol_EdgeCases(t *testing.T) {
	t.Run("symbol with spaces", func(t *testing.T) {
		symbol := &Symbol{
			Name:       "BTC USDT", // Space in name
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
			Status:     SymbolStatusTrading,
		}
		// Validation allows spaces
		assert.NoError(t, symbol.Validate())
	})

	t.Run("symbol with special characters", func(t *testing.T) {
		symbol := &Symbol{
			Name:       "BTC/USDT", // Slash in name
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
			Status:     SymbolStatusTrading,
		}
		// Validation allows special characters
		assert.NoError(t, symbol.Validate())
	})

	t.Run("very long symbol name", func(t *testing.T) {
		symbol := &Symbol{
			Name:       "VERYLONGBASEASSETNAMEUSDT",
			BaseAsset:  "VERYLONGBASEASSSETNAME",
			QuoteAsset: "USDT",
			Status:     SymbolStatusTrading,
		}
		assert.NoError(t, symbol.Validate())
	})
}
