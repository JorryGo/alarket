package trader

import (
	"alarket/internal/binance/processors"
	"fmt"
	"sync"
)

type Price struct {
	Symbol string
	Price  float64
	mu     sync.RWMutex
}

func (p *Price) setPrice(price float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Price = price
}

func (p *Price) getPrice() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Price
}

type Trader struct {
	priceStorage map[string]*Price
	tradingTree  map[string]*processors.SymbolTree
	mu           sync.RWMutex
}

func InitTrader(tree *map[string]*processors.SymbolTree) *Trader {
	return &Trader{priceStorage: make(map[string]*Price), tradingTree: *tree}
}

func (t *Trader) SetPrice(symbol string, price float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	priceStruct, ok := t.priceStorage[symbol]

	if !ok {
		t.priceStorage[symbol] = &Price{Symbol: symbol, Price: price}
		return
	}

	priceStruct.setPrice(price)
}

func (t *Trader) Price(symbol string) (float64, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	price, ok := t.priceStorage[symbol]
	if !ok {
		return 0, false
	}

	return price.getPrice(), true
}

func (t *Trader) CheckLoopDiffs(symbol string) {
	rootTicker, ok := t.tradingTree[symbol]
	if !ok {
		return
	}

	fmt.Println("CheckLoopDiffs", rootTicker)
}
