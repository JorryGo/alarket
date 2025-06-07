package dto

type TradeEventDTO struct {
	EventType     string    `json:"e"`
	EventTime     int64     `json:"E"`
	Symbol        string    `json:"s"`
	TradeID       int64     `json:"t"`
	Price         string    `json:"p"`
	Quantity      string    `json:"q"`
	BuyerOrderID  int64     `json:"b"`
	SellerOrderID int64     `json:"a"`
	TradeTime     int64     `json:"T"`
	IsBuyerMaker  bool      `json:"m"`
}

type BookTickerEventDTO struct {
	EventType       string `json:"e"`
	UpdateID        int64  `json:"u"`
	Symbol          string `json:"s"`
	BestBidPrice    string `json:"b"`
	BestBidQuantity string `json:"B"`
	BestAskPrice    string `json:"a"`
	BestAskQuantity string `json:"A"`
	TransactionTime int64  `json:"T"`
	EventTime       int64  `json:"E"`
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