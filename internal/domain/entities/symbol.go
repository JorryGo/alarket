package entities

type SymbolStatus string

const (
	SymbolStatusTrading      SymbolStatus = "TRADING"
	SymbolStatusHalt         SymbolStatus = "HALT"
	SymbolStatusBreak        SymbolStatus = "BREAK"
	SymbolStatusAuctionMatch SymbolStatus = "AUCTION_MATCH"
	SymbolStatusEndOfDay     SymbolStatus = "END_OF_DAY"
	SymbolStatusPreTrading   SymbolStatus = "PRE_TRADING"
	SymbolStatusPostTrading  SymbolStatus = "POST_TRADING"
)

type Symbol struct {
	Name            string
	BaseAsset       string
	QuoteAsset      string
	Status          SymbolStatus
	IsSpotTrading   bool
	IsMarginTrading bool
}

func NewSymbol(name, baseAsset, quoteAsset string, status SymbolStatus) *Symbol {
	return &Symbol{
		Name:       name,
		BaseAsset:  baseAsset,
		QuoteAsset: quoteAsset,
		Status:     status,
	}
}

func (s *Symbol) IsActive() bool {
	return s.Status == SymbolStatusTrading
}

func (s *Symbol) Validate() error {
	if s.Name == "" {
		return ErrInvalidSymbol
	}
	if s.BaseAsset == "" || s.QuoteAsset == "" {
		return ErrInvalidAsset
	}
	return nil
}
