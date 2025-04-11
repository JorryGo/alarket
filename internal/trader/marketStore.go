package trader

import (
	"alarket/internal/binance/processors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
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
	locked       atomic.Bool
	pricerMu     sync.Mutex
}

func (t *Trader) tryLock() bool {
	if !t.locked.CompareAndSwap(false, true) {
		return false // уже заблокировано
	}
	t.pricerMu.Lock()
	return true
}

func (t *Trader) tunlock() {
	t.locked.Store(false)
	t.pricerMu.Unlock()
}

func InitTrader(tree *map[string]*processors.SymbolTree) *Trader {
	return &Trader{priceStorage: make(map[string]*Price), tradingTree: *tree}
}

func (t *Trader) SetPrice(symbol string, price float64) {
	t.mu.RLock()
	priceStruct, ok := t.priceStorage[symbol]
	t.mu.RUnlock()

	if !ok {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.priceStorage[symbol] = &Price{
			Symbol:      symbol,
			Price:       price,
			LastUpdated: time.Now(),
		}
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
	if time.Since(price.LastUpdated) > time.Second*2 {
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

	if !t.tryLock() {
		return
	}

	defer t.tunlock()
	fmt.Print("\033[2J\033[H")

	start := time.Now()
	out := make([]string, 0, 7)

	for _, secondNode := range *node.To {
		startedMoney := initialAmount
		currentAmount := initialAmount

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
				currentAmount *= firstPrice
			} else {
				ownerOfCoin = node.Symbol.BaseAsset
				currentAmount /= firstPrice
			}

			if secondNode.Symbol.BaseAsset == ownerOfCoin {
				ownerOfCoin = secondNode.Symbol.QuoteAsset
				currentAmount *= secondPrice
			} else {
				ownerOfCoin = secondNode.Symbol.BaseAsset
				currentAmount /= secondPrice
			}

			if lastNode.Symbol.BaseAsset == ownerOfCoin {
				currentAmount *= lastPrice
			} else {
				currentAmount /= lastPrice
			}

			aa := ((currentAmount / startedMoney) - 1) * 100

			//fmt.Println(path, currentAmount, aa)

			if aa > 2 {
				out = append(out, fmt.Sprintf("%s %f %f", path, currentAmount, aa))
				//fmt.Println(path, currentAmount, aa)
			}
		}
	}

	sort.Strings(out)
	fmt.Println(strings.Join(out, "\n"))
	duration := time.Since(start) // вычисляем разницу
	fmt.Printf("%s - Время выполнения: %s\n", time.Now(), duration)

}
