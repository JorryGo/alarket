package dto

type TradeEventDTO struct {
	EventType          string `json:"e"`
	EventTime          int64  `json:"E"`
	Symbol             string `json:"s"`
	TradeID            int64  `json:"t"`
	Price              string `json:"p"`
	Quantity           string `json:"q"`
	TradeTime          int64  `json:"T"`
	IsBuyerMarketMaker bool   `json:"m"`
	Ignore             bool   `json:"M"` // Ignore the uppercase M field
}

type BookTickerEventDTO struct {
	UpdateID        int64  `json:"u"`
	Symbol          string `json:"s"`
	BestBidPrice    string `json:"b"`
	BestBidQuantity string `json:"B"`
	BestAskPrice    string `json:"a"`
	BestAskQuantity string `json:"A"`
}

type SubscriptionRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     int      `json:"id"`
}

type SubscriptionResponse struct {
	Result interface{} `json:"result"`
	ID     int         `json:"id"`
}
