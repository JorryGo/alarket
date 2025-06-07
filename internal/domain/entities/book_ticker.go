package entities

import (
	"time"
)

type BookTicker struct {
	UpdateID        int64
	Symbol          string
	BestBidPrice    float64
	BestBidQuantity float64
	BestAskPrice    float64
	BestAskQuantity float64
	TransactionTime time.Time
	EventTime       time.Time
}

func NewBookTicker(
	updateID int64,
	symbol string,
	bestBidPrice float64,
	bestBidQuantity float64,
	bestAskPrice float64,
	bestAskQuantity float64,
	transactionTime time.Time,
	eventTime time.Time,
) *BookTicker {
	return &BookTicker{
		UpdateID:        updateID,
		Symbol:          symbol,
		BestBidPrice:    bestBidPrice,
		BestBidQuantity: bestBidQuantity,
		BestAskPrice:    bestAskPrice,
		BestAskQuantity: bestAskQuantity,
		TransactionTime: transactionTime,
		EventTime:       eventTime,
	}
}

func (b *BookTicker) Validate() error {
	if b.Symbol == "" {
		return ErrInvalidSymbol
	}
	if b.BestBidPrice < 0 || b.BestAskPrice < 0 {
		return ErrInvalidPrice
	}
	if b.BestBidQuantity < 0 || b.BestAskQuantity < 0 {
		return ErrInvalidQuantity
	}
	if b.BestBidPrice > b.BestAskPrice && b.BestAskPrice > 0 {
		return ErrInvalidSpread
	}
	return nil
}
