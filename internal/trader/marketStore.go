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
	BidPrice    float64
	AskPrice    float64
	LastUpdated time.Time
	mu          sync.RWMutex
}

func (p *Price) setPrice(bidPrice float64, askPrice float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.BidPrice = bidPrice
	p.AskPrice = askPrice
	p.LastUpdated = time.Now()
}

func (p *Price) getBidPrice() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.BidPrice
}

func (p *Price) getAskPrice() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.AskPrice
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

func (t *Trader) SetPrice(symbol string, bidPrice float64, askPrice float64) {
	t.mu.RLock()
	priceStruct, ok := t.priceStorage[symbol]
	t.mu.RUnlock()

	if !ok {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.priceStorage[symbol] = &Price{
			Symbol:      symbol,
			BidPrice:    bidPrice,
			AskPrice:    askPrice,
			LastUpdated: time.Now(),
		}
		return
	}

	priceStruct.setPrice(bidPrice, askPrice)
}

func (t *Trader) BidPrice(symbol string) (float64, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	price, ok := t.priceStorage[symbol]

	if !ok {
		return 0, false
	}

	// Check if more than 1 second has passed since the last update
	if time.Since(price.LastUpdated) > time.Hour {
		return 0, false
	}

	return price.getBidPrice(), true
}

func (t *Trader) AskPrice(symbol string) (float64, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	price, ok := t.priceStorage[symbol]

	if !ok {
		return 0, false
	}

	// Check if more than 1 second has passed since the last update
	if time.Since(price.LastUpdated) > time.Hour {
		return 0, false
	}

	return price.getAskPrice(), true
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

	start := time.Now()
	out := make([]string, 0, 7)

	for _, secondNode := range *node.To {
		startedMoney := initialAmount
		currentAmount := initialAmount

		for _, lastNode := range *secondNode.To {
			path := node.SymbolName + " -> " + secondNode.SymbolName + " -> " + lastNode.SymbolName

			// Первая сделка: BTC/USDT - покупаем BTC за USDT, используем Ask цену
			firstPrice, ok := t.AskPrice(node.SymbolName)
			if !ok {
				continue
			}

			// Определим, что у нас сейчас есть (после первой сделки)
			var ownerOfCoin string

			// После покупки BTCUSDT у нас на руках BTC
			ownerOfCoin = node.Symbol.BaseAsset
			currentAmount /= firstPrice

			// Вторая сделка: определяем направление и какую цену использовать
			var secondPrice float64
			if secondNode.Symbol.BaseAsset == ownerOfCoin {
				// Мы продаем наш ownerOfCoin (например, BTC), используем Bid цену
				secondPrice, ok = t.BidPrice(secondNode.SymbolName)
				ownerOfCoin = secondNode.Symbol.QuoteAsset
				currentAmount *= secondPrice
			} else {
				// Мы покупаем базовый актив за наш ownerOfCoin, используем Ask цену
				secondPrice, ok = t.AskPrice(secondNode.SymbolName)
				ownerOfCoin = secondNode.Symbol.BaseAsset
				currentAmount /= secondPrice
			}
			if !ok {
				continue
			}

			// Третья сделка: последний шаг треугольника
			var lastPrice float64
			if lastNode.Symbol.BaseAsset == ownerOfCoin {
				// Мы продаем наш ownerOfCoin, используем Bid цену
				lastPrice, ok = t.BidPrice(lastNode.SymbolName)
				currentAmount *= lastPrice
			} else {
				// Мы покупаем базовый актив за наш ownerOfCoin, используем Ask цену
				lastPrice, ok = t.AskPrice(lastNode.SymbolName)
				currentAmount /= lastPrice
			}
			if !ok {
				continue
			}

			aa := ((currentAmount / startedMoney) - 1) * 100

			//fmt.Println(path, currentAmount, aa)

			if aa > 0.3 {
				out = append(out, fmt.Sprintf("%s %f %f", path, currentAmount, aa))
				//fmt.Println(path, currentAmount, aa)
			}
		}
	}

	if len(out) == 0 {
		return
	}

	sort.Strings(out)
	fmt.Println(strings.Join(out, "\n"))
	duration := time.Since(start) // вычисляем разницу
	fmt.Printf("%s - Время выполнения: %s\n", time.Now(), duration)
}
