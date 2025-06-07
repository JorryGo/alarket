package entities

import (
	"time"
)

type Trade struct {
	ID           string
	Symbol       string
	Price        float64
	Quantity     float64
	Time         time.Time
	IsBuyerMaker bool
	EventTime    time.Time
}

func NewTrade(
	id string,
	symbol string,
	price float64,
	quantity float64,
	tradeTime time.Time,
	isBuyerMaker bool,
	eventTime time.Time,
) *Trade {
	return &Trade{
		ID:           id,
		Symbol:       symbol,
		Price:        price,
		Quantity:     quantity,
		Time:         tradeTime,
		IsBuyerMaker: isBuyerMaker,
		EventTime:    eventTime,
	}
}

func (t *Trade) Validate() error {
	if t.Symbol == "" {
		return ErrInvalidSymbol
	}
	if t.Price <= 0 {
		return ErrInvalidPrice
	}
	if t.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	return nil
}
