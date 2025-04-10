package trader

import (
	"alarket/internal/binance/processors"
	"fmt"
	"sync"
	"time"
)

type Price struct {
	Symbol      string
	Price       float64
	LastUpdated time.Time
	mu          sync.RWMutex
}

func (p *Price) setPrice(price float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Price = price
	p.LastUpdated = time.Now()
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

	// Check if more than 1 second has passed since the last update
	if time.Since(price.LastUpdated) > time.Second*10 {
		return 0, false
	}

	return price.getPrice(), true
}

func (t *Trader) CheckLoopDiffs(symbol string) {
	rootTicker, ok := t.tradingTree[symbol]
	if !ok {
		return
	}

	// Calculate price differences across the trading loop
	t.checkLoop(rootTicker, 1.0)
}

// checkLoop recursively checks trading loops, computing the price difference
// initialAmount is the theoretical amount we start with (1.0 unit)
// path is the current trading path for logging
func (t *Trader) checkLoop(node *processors.SymbolTree, initialAmount float64) {
	if node.To == nil || len(*node.To) == 0 {
		return
	}

	startedMoney := initialAmount

	for _, secondNode := range *node.To {
		for _, lastNode := range *secondNode.To {
			path := node.SymbolName + " -> " + secondNode.SymbolName + " -> " + lastNode.SymbolName

			firstPrice, ok := t.Price(node.SymbolName)
			if !ok {
				continue
			}
			secondPrice, ok := t.Price(secondNode.SymbolName)
			if !ok {
				continue
			}
			lastPrice, ok := t.Price(lastNode.SymbolName)
			if !ok {
				continue
			}

			var ownerOfCoin string

			if node.Symbol.BaseAsset == "USDT" {
				ownerOfCoin = node.Symbol.QuoteAsset
				initialAmount *= firstPrice
			} else {
				ownerOfCoin = node.Symbol.BaseAsset
				initialAmount /= firstPrice
			}

			if secondNode.Symbol.BaseAsset == ownerOfCoin {
				ownerOfCoin = secondNode.Symbol.QuoteAsset
				initialAmount *= secondPrice
			} else {
				ownerOfCoin = secondNode.Symbol.BaseAsset
				initialAmount /= secondPrice
			}

			if lastNode.Symbol.BaseAsset == ownerOfCoin {
				initialAmount *= lastPrice
			} else {
				initialAmount /= lastPrice
			}

			aa := 100 - (initialAmount / startedMoney * 100)

			fmt.Println(path, initialAmount, aa)

			//if (aa > 0.5 || aa < -0.5) && initialAmount > 1.001 {
			//	fmt.Println(path, initialAmount, aa)
			//}
		}
	}

}
