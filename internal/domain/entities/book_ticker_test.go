package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBookTicker(t *testing.T) {
	now := time.Now()
	eventTime := now.Add(time.Millisecond * 100)

	bt := NewBookTicker(
		123456,
		"BTCUSDT",
		49999.0,
		1.5,
		50000.0,
		2.0,
		now,
		eventTime,
	)

	assert.NotNil(t, bt)
	assert.Equal(t, int64(123456), bt.UpdateID)
	assert.Equal(t, "BTCUSDT", bt.Symbol)
	assert.Equal(t, 49999.0, bt.BestBidPrice)
	assert.Equal(t, 1.5, bt.BestBidQuantity)
	assert.Equal(t, 50000.0, bt.BestAskPrice)
	assert.Equal(t, 2.0, bt.BestAskQuantity)
	assert.Equal(t, now, bt.TransactionTime)
	assert.Equal(t, eventTime, bt.EventTime)
}

func TestBookTicker_Validate(t *testing.T) {
	tests := []struct {
		name    string
		bt      *BookTicker
		wantErr error
	}{
		{
			name: "valid book ticker",
			bt: &BookTicker{
				UpdateID:        123456,
				Symbol:          "BTCUSDT",
				BestBidPrice:    49999.0,
				BestBidQuantity: 1.5,
				BestAskPrice:    50000.0,
				BestAskQuantity: 2.0,
				TransactionTime: time.Now(),
				EventTime:       time.Now(),
			},
			wantErr: nil,
		},
		{
			name: "empty symbol",
			bt: &BookTicker{
				UpdateID:        123456,
				Symbol:          "",
				BestBidPrice:    49999.0,
				BestBidQuantity: 1.5,
				BestAskPrice:    50000.0,
				BestAskQuantity: 2.0,
			},
			wantErr: ErrInvalidSymbol,
		},
		{
			name: "negative bid price",
			bt: &BookTicker{
				UpdateID:        123456,
				Symbol:          "BTCUSDT",
				BestBidPrice:    -100.0,
				BestBidQuantity: 1.5,
				BestAskPrice:    50000.0,
				BestAskQuantity: 2.0,
			},
			wantErr: ErrInvalidPrice,
		},
		{
			name: "negative ask price",
			bt: &BookTicker{
				UpdateID:        123456,
				Symbol:          "BTCUSDT",
				BestBidPrice:    49999.0,
				BestBidQuantity: 1.5,
				BestAskPrice:    -50000.0,
				BestAskQuantity: 2.0,
			},
			wantErr: ErrInvalidPrice,
		},
		{
			name: "negative bid quantity",
			bt: &BookTicker{
				UpdateID:        123456,
				Symbol:          "BTCUSDT",
				BestBidPrice:    49999.0,
				BestBidQuantity: -1.5,
				BestAskPrice:    50000.0,
				BestAskQuantity: 2.0,
			},
			wantErr: ErrInvalidQuantity,
		},
		{
			name: "negative ask quantity",
			bt: &BookTicker{
				UpdateID:        123456,
				Symbol:          "BTCUSDT",
				BestBidPrice:    49999.0,
				BestBidQuantity: 1.5,
				BestAskPrice:    50000.0,
				BestAskQuantity: -2.0,
			},
			wantErr: ErrInvalidQuantity,
		},
		{
			name: "invalid spread - bid higher than ask",
			bt: &BookTicker{
				UpdateID:        123456,
				Symbol:          "BTCUSDT",
				BestBidPrice:    50001.0,
				BestBidQuantity: 1.5,
				BestAskPrice:    50000.0,
				BestAskQuantity: 2.0,
			},
			wantErr: ErrInvalidSpread,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.bt.Validate()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBookTicker_EdgeCases(t *testing.T) {
	t.Run("zero prices and quantities are valid", func(t *testing.T) {
		bt := &BookTicker{
			Symbol:          "BTCUSDT",
			BestBidPrice:    0,
			BestBidQuantity: 0,
			BestAskPrice:    0,
			BestAskQuantity: 0,
		}
		assert.NoError(t, bt.Validate())
	})

	t.Run("bid equals ask is valid", func(t *testing.T) {
		bt := &BookTicker{
			Symbol:          "BTCUSDT",
			BestBidPrice:    50000.0,
			BestBidQuantity: 1.0,
			BestAskPrice:    50000.0,
			BestAskQuantity: 1.0,
		}
		assert.NoError(t, bt.Validate())
	})

	t.Run("zero ask price allows any bid price", func(t *testing.T) {
		bt := &BookTicker{
			Symbol:          "BTCUSDT",
			BestBidPrice:    100000.0,
			BestBidQuantity: 1.0,
			BestAskPrice:    0, // Market might have no asks
			BestAskQuantity: 0,
		}
		assert.NoError(t, bt.Validate())
	})

	t.Run("very small spread is valid", func(t *testing.T) {
		bt := &BookTicker{
			Symbol:          "BTCUSDT",
			BestBidPrice:    49999.99999999,
			BestBidQuantity: 1.0,
			BestAskPrice:    50000.00000001,
			BestAskQuantity: 1.0,
		}
		assert.NoError(t, bt.Validate())
	})

	t.Run("zero time values", func(t *testing.T) {
		bt := NewBookTicker(
			123456,
			"BTCUSDT",
			49999.0,
			1.5,
			50000.0,
			2.0,
			time.Time{}, // zero time
			time.Time{}, // zero time
		)
		assert.NotNil(t, bt)
		assert.True(t, bt.TransactionTime.IsZero())
		assert.True(t, bt.EventTime.IsZero())
		// Validation doesn't check time values
		assert.NoError(t, bt.Validate())
	})
}
