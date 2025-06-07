package events

import "alarket/internal/domain/entities"

type EventType string

const (
	TradeEventType      EventType = "trade"
	BookTickerEventType EventType = "bookTicker"
)

type DomainEvent interface {
	Type() EventType
}

type TradeEvent struct {
	Trade *entities.Trade
}

func (e TradeEvent) Type() EventType {
	return TradeEventType
}

type BookTickerEvent struct {
	BookTicker *entities.BookTicker
}

func (e BookTickerEvent) Type() EventType {
	return BookTickerEventType
}