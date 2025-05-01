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
	executor     *Executor
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

func InitTrader(tree *map[string]*processors.SymbolTree, executor *Executor) *Trader {
	return &Trader{priceStorage: make(map[string]*Price), tradingTree: *tree, executor: executor}
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

// executeTrades executes trades along a chain when potential growth is above threshold
// Returns true if all trades were executed successfully, false otherwise
func (t *Trader) executeTrades(path string, node *processors.SymbolTree, secondNode *processors.SymbolTree, lastNode *processors.SymbolTree, ownerOfCoin string) bool {
	// Execute trades along the chain
	fmt.Printf("Executing trades for path: %s\n", path)

	var err error

	// Parse the path to get the symbols
	pathParts := strings.Split(path, " -> ")
	if len(pathParts) != 3 {
		fmt.Printf("Invalid path format: %s\n", path)
		return false
	}

	// Initial amount in USDT
	initialUSDT := 20.0 // Starting with 100 USDT for example
	currentUSDT := initialUSDT
	fmt.Printf("Starting with %.2f USDT\n", currentUSDT)

	// First trade: USDT -> First asset
	firstSymbol := pathParts[0]
	var firstReport ExecutionReport

	// Check if USDT is in BaseAsset or QuoteAsset position
	if node.Symbol.BaseAsset == "USDT" {
		// If USDT is the base asset, we need to sell USDT to get the quote asset
		// For selling, the quantity is in terms of the base asset (USDT), so we can use currentUSDT directly
		fmt.Printf("Executing first trade: %s, selling %.2f USDT\n", firstSymbol, currentUSDT)
		firstReport, err = t.executor.SellMarket(firstSymbol, currentUSDT)
	} else {
		// If USDT is the quote asset, we need to buy the base asset with USDT
		// For buying, the quantity must be in terms of the base asset, not USDT
		// We need to calculate how much of the base asset we can buy with our USDT
		askPrice, ok := t.AskPrice(firstSymbol)
		if !ok {
			fmt.Printf("Error getting ask price for %s\n", firstSymbol)
			return false
		}

		// Calculate the quantity of the base asset we can buy with our USDT
		baseAssetQuantity := currentUSDT / askPrice

		fmt.Printf("Executing first trade: %s, buying %.8f %s with %.2f USDT (price: %.8f)\n",
			firstSymbol, baseAssetQuantity, node.Symbol.BaseAsset, currentUSDT, askPrice)
		firstReport, err = t.executor.BuyMarket(firstSymbol, baseAssetQuantity)
	}

	if err != nil {
		fmt.Printf("Error executing first trade: %v\n", err)
		return false
	}

	// Update current amount based on the execution report
	currentAmount := firstReport.CumQty

	// After first trade, determine which asset we own based on the trade direction
	if node.Symbol.BaseAsset == "USDT" {
		// If we sold USDT (base asset), we now own the quote asset
		ownerOfCoin = node.Symbol.QuoteAsset
	} else {
		// If we bought with USDT (quote asset), we now own the base asset
		ownerOfCoin = node.Symbol.BaseAsset
	}

	fmt.Printf("First trade executed: Got %.8f %s\n", currentAmount, ownerOfCoin)

	// Second trade
	secondSymbol := pathParts[1]
	var secondReport ExecutionReport

	if secondNode.Symbol.BaseAsset == ownerOfCoin {
		// We're selling our asset (which is the base asset)
		// For selling, the quantity is in terms of the base asset, so we can use currentAmount directly
		fmt.Printf("Executing second trade: %s, selling %.8f %s\n", secondSymbol, currentAmount, ownerOfCoin)
		secondReport, err = t.executor.SellMarket(secondSymbol, currentAmount)
		// After selling, we now own the quote asset
		ownerOfCoin = secondNode.Symbol.QuoteAsset
	} else {
		// We're buying with our asset (which is the quote asset)
		// For buying, the quantity must be in terms of the base asset, not the quote asset
		// We need to calculate how much of the base asset we can buy with our quote asset
		askPrice, ok := t.AskPrice(secondSymbol)
		if !ok {
			fmt.Printf("Error getting ask price for %s\n", secondSymbol)
			return false
		}

		// Calculate the quantity of the base asset we can buy with our quote asset
		baseAssetQuantity := currentAmount / askPrice

		fmt.Printf("Executing second trade: %s, buying %.8f %s with %.8f %s (price: %.8f)\n",
			secondSymbol, baseAssetQuantity, secondNode.Symbol.BaseAsset, currentAmount, ownerOfCoin, askPrice)
		secondReport, err = t.executor.BuyMarket(secondSymbol, baseAssetQuantity)
		// After buying, we now own the base asset
		ownerOfCoin = secondNode.Symbol.BaseAsset
	}

	if err != nil {
		fmt.Printf("Error executing second trade: %v\n", err)
		return false
	}

	// Update current amount based on the execution report
	currentAmount = secondReport.CumQty
	fmt.Printf("Second trade executed: Got %.8f %s\n", currentAmount, ownerOfCoin)

	// Third trade
	lastSymbol := pathParts[2]
	var lastReport ExecutionReport

	if lastNode.Symbol.BaseAsset == ownerOfCoin {
		// We're selling our asset (which is the base asset)
		// For selling, the quantity is in terms of the base asset, so we can use currentAmount directly
		fmt.Printf("Executing third trade: %s, selling %.8f %s\n", lastSymbol, currentAmount, ownerOfCoin)
		lastReport, err = t.executor.SellMarket(lastSymbol, currentAmount)
	} else {
		// We're buying with our asset (which is the quote asset)
		// For buying, the quantity must be in terms of the base asset, not the quote asset
		// We need to calculate how much of the base asset we can buy with our quote asset
		askPrice, ok := t.AskPrice(lastSymbol)
		if !ok {
			fmt.Printf("Error getting ask price for %s\n", lastSymbol)
			return false
		}

		// Calculate the quantity of the base asset we can buy with our quote asset
		baseAssetQuantity := currentAmount / askPrice

		fmt.Printf("Executing third trade: %s, buying %.8f %s with %.8f %s (price: %.8f)\n",
			lastSymbol, baseAssetQuantity, lastNode.Symbol.BaseAsset, currentAmount, ownerOfCoin, askPrice)
		lastReport, err = t.executor.BuyMarket(lastSymbol, baseAssetQuantity)
	}

	if err != nil {
		fmt.Printf("Error executing third trade: %v\n", err)
		return false
	}

	// Final amount in USDT
	finalUSDT := lastReport.CumQuoteQty
	profit := finalUSDT - initialUSDT
	profitPercentage := (profit / initialUSDT) * 100

	fmt.Printf("Trades completed! Initial: %.2f USDT, Final: %.2f USDT\n", initialUSDT, finalUSDT)
	fmt.Printf("Profit: %.2f USDT (%.2f%%)\n", profit, profitPercentage)

	return true
}

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

			potentialGrowth := ((currentAmount / startedMoney) - 1) * 100

			//fmt.Println(path, currentAmount)

			if potentialGrowth > 0.0 {
				out = append(out, fmt.Sprintf("%s %f", path, currentAmount))
				//fmt.Println(path, currentAmount)

				// Execute trades along the chain
				if !t.executeTrades(path, node, secondNode, lastNode, "USDT") {
					continue
				}
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
