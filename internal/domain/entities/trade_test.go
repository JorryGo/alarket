package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTrade(t *testing.T) {
	now := time.Now()
	eventTime := now.Add(time.Millisecond * 100)

	trade := NewTrade(
		"12345",
		"BTCUSDT",
		50000.0,
		0.01,
		now,
		true,
		eventTime,
	)

	assert.NotNil(t, trade)
	assert.Equal(t, "12345", trade.ID)
	assert.Equal(t, "BTCUSDT", trade.Symbol)
	assert.Equal(t, 50000.0, trade.Price)
	assert.Equal(t, 0.01, trade.Quantity)
	assert.Equal(t, now, trade.Time)
	assert.True(t, trade.IsBuyerMaker)
	assert.Equal(t, eventTime, trade.EventTime)
}

func TestTrade_Validate(t *testing.T) {
	tests := []struct {
		name    string
		trade   *Trade
		wantErr error
	}{
		{
			name: "valid trade",
			trade: &Trade{
				ID:           "12345",
				Symbol:       "BTCUSDT",
				Price:        50000.0,
				Quantity:     0.01,
				Time:         time.Now(),
				IsBuyerMaker: true,
				EventTime:    time.Now(),
			},
			wantErr: nil,
		},
		{
			name: "empty symbol",
			trade: &Trade{
				ID:           "12345",
				Symbol:       "",
				Price:        50000.0,
				Quantity:     0.01,
				Time:         time.Now(),
				IsBuyerMaker: true,
				EventTime:    time.Now(),
			},
			wantErr: ErrInvalidSymbol,
		},
		{
			name: "zero price",
			trade: &Trade{
				ID:           "12345",
				Symbol:       "BTCUSDT",
				Price:        0,
				Quantity:     0.01,
				Time:         time.Now(),
				IsBuyerMaker: true,
				EventTime:    time.Now(),
			},
			wantErr: ErrInvalidPrice,
		},
		{
			name: "negative price",
			trade: &Trade{
				ID:           "12345",
				Symbol:       "BTCUSDT",
				Price:        -100.0,
				Quantity:     0.01,
				Time:         time.Now(),
				IsBuyerMaker: true,
				EventTime:    time.Now(),
			},
			wantErr: ErrInvalidPrice,
		},
		{
			name: "zero quantity",
			trade: &Trade{
				ID:           "12345",
				Symbol:       "BTCUSDT",
				Price:        50000.0,
				Quantity:     0,
				Time:         time.Now(),
				IsBuyerMaker: true,
				EventTime:    time.Now(),
			},
			wantErr: ErrInvalidQuantity,
		},
		{
			name: "negative quantity",
			trade: &Trade{
				ID:           "12345",
				Symbol:       "BTCUSDT",
				Price:        50000.0,
				Quantity:     -0.01,
				Time:         time.Now(),
				IsBuyerMaker: true,
				EventTime:    time.Now(),
			},
			wantErr: ErrInvalidQuantity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trade.Validate()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTrade_EdgeCases(t *testing.T) {
	t.Run("very small valid quantity", func(t *testing.T) {
		trade := &Trade{
			Symbol:   "BTCUSDT",
			Price:    50000.0,
			Quantity: 0.00000001, // 1 satoshi worth
		}
		assert.NoError(t, trade.Validate())
	})

	t.Run("very large valid price", func(t *testing.T) {
		trade := &Trade{
			Symbol:   "BTCUSDT",
			Price:    1e9, // 1 billion
			Quantity: 0.01,
		}
		assert.NoError(t, trade.Validate())
	})

	t.Run("zero time values", func(t *testing.T) {
		trade := NewTrade(
			"12345",
			"BTCUSDT",
			50000.0,
			0.01,
			time.Time{}, // zero time
			true,
			time.Time{}, // zero time
		)
		assert.NotNil(t, trade)
		assert.True(t, trade.Time.IsZero())
		assert.True(t, trade.EventTime.IsZero())
		// Validation doesn't check time values
		assert.NoError(t, trade.Validate())
	})
}
