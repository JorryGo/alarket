package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"alarket/internal/application/dto"
	"alarket/internal/application/usecases"
	"alarket/internal/domain/entities"
)

type EventHandler struct {
	processTradeUC      *usecases.ProcessTradeEventUseCase
	processBookTickerUC *usecases.ProcessBookTickerEventUseCase
	logger              *slog.Logger
}

func NewEventHandler(
	processTradeUC *usecases.ProcessTradeEventUseCase,
	processBookTickerUC *usecases.ProcessBookTickerEventUseCase,
	logger *slog.Logger,
) *EventHandler {
	return &EventHandler{
		processTradeUC:      processTradeUC,
		processBookTickerUC: processBookTickerUC,
		logger:              logger,
	}
}

func (h *EventHandler) HandleMessage(ctx context.Context, message []byte) error {
	var baseEvent map[string]interface{}
	if err := json.Unmarshal(message, &baseEvent); err != nil {
		h.logger.Error("Failed to parse message", "error", err)
		return err
	}

	// Check if it's a trade event (has "e" field)
	eventType, hasEventType := baseEvent["e"].(string)
	if hasEventType {
		switch eventType {
		case "trade":
			return h.handleTradeEvent(ctx, message)
		default:
			h.logger.Debug("Unknown event type", "type", eventType)
			return nil
		}
	}

	// Check if it's a book ticker (has "u" field for updateId)
	if _, isBookTicker := baseEvent["u"]; isBookTicker {
		return h.handleBookTickerEvent(ctx, message)
	}

	h.logger.Debug("Received non-event message", "message", string(message))
	return nil
}

func (h *EventHandler) handleTradeEvent(ctx context.Context, message []byte) error {
	var event dto.TradeEventDTO
	if err := json.Unmarshal(message, &event); err != nil {
		h.logger.Error("Failed to parse trade event", "error", err)
		return err
	}

	price, err := strconv.ParseFloat(event.Price, 64)
	if err != nil {
		return fmt.Errorf("invalid price: %w", err)
	}

	quantity, err := strconv.ParseFloat(event.Quantity, 64)
	if err != nil {
		return fmt.Errorf("invalid quantity: %w", err)
	}

	trade := entities.NewTrade(
		strconv.FormatInt(event.TradeID, 10),
		event.Symbol,
		price,
		quantity,
		0, // BuyerOrderID not provided in trade stream
		0, // SellerOrderID not provided in trade stream
		time.UnixMilli(event.TradeTime),
		event.IsBuyerMarketMaker,
		time.UnixMilli(event.EventTime),
	)

	return h.processTradeUC.Execute(ctx, trade)
}

func (h *EventHandler) handleBookTickerEvent(ctx context.Context, message []byte) error {
	var event dto.BookTickerEventDTO
	if err := json.Unmarshal(message, &event); err != nil {
		h.logger.Error("Failed to parse book ticker event", "error", err)
		return err
	}

	bidPrice, err := strconv.ParseFloat(event.BestBidPrice, 64)
	if err != nil {
		return fmt.Errorf("invalid bid price: %w", err)
	}

	bidQuantity, err := strconv.ParseFloat(event.BestBidQuantity, 64)
	if err != nil {
		return fmt.Errorf("invalid bid quantity: %w", err)
	}

	askPrice, err := strconv.ParseFloat(event.BestAskPrice, 64)
	if err != nil {
		return fmt.Errorf("invalid ask price: %w", err)
	}

	askQuantity, err := strconv.ParseFloat(event.BestAskQuantity, 64)
	if err != nil {
		return fmt.Errorf("invalid ask quantity: %w", err)
	}

	// Book ticker events don't have timestamps, so we use current time
	now := time.Now()
	bookTicker := entities.NewBookTicker(
		event.UpdateID,
		event.Symbol,
		bidPrice,
		bidQuantity,
		askPrice,
		askQuantity,
		now, // transaction time
		now, // event time
	)

	return h.processBookTickerUC.Execute(ctx, bookTicker)
}
