package trader

import (
	"alarket/internal/binance/processors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"
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

// adjustQuantityToLotSize adjusts the quantity to meet the minimum lot size requirement
// and ensures it's a valid multiple of the lot size
func (t *Trader) adjustQuantityToLotSize(symbol string, quantity float64) float64 {
	// Find the symbol in the trading tree
	symbolTree, ok := t.tradingTree[symbol]
	if !ok {
		// If symbol not found, return the original quantity
		fmt.Printf("Symbol %s not found in trading tree, using original quantity\n", symbol)
		return quantity
	}

	// Get the minimum lot size
	minLotSize := symbolTree.LotMinQty
	if minLotSize <= 0 {
		// If minimum lot size is invalid, return the original quantity
		fmt.Printf("Invalid minimum lot size for %s: %f, using original quantity\n", symbol, minLotSize)
		return quantity
	}

	// Convert to decimal for precise calculations
	decQuantity := decimal.NewFromFloat(quantity)
	decMinLotSize := decimal.NewFromFloat(minLotSize)

	// Adjust quantity to be a multiple of the minimum lot size
	// First, ensure it's at least the minimum lot size
	if decQuantity.LessThan(decMinLotSize) {
		fmt.Printf("Adjusting quantity from %f to minimum lot size %f for %s\n", quantity, minLotSize, symbol)
		return minLotSize
	}

	// Calculate how many complete lot sizes fit into the quantity
	// Use integer division to get the floor value
	lotSizeMultiple := decQuantity.Div(decMinLotSize).Floor()
	adjustedQuantity := lotSizeMultiple.Mul(decMinLotSize)

	// Convert back to float64 for return
	adjustedFloat, _ := adjustedQuantity.Float64()

	if adjustedFloat != quantity {
		fmt.Printf("Adjusting quantity from %f to %f for %s (lot size: %f)\n",
			quantity, adjustedFloat, symbol, minLotSize)
	}

	return adjustedFloat
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
	initialUSDT := 20.0 // Starting with 20 USDT by default
	currentUSDT := initialUSDT
	fmt.Printf("Starting with %.2f USDT\n", currentUSDT)

	// First trade: USDT -> First asset
	firstSymbol := pathParts[0]
	var firstReport ExecutionReport

	// Check if USDT is in BaseAsset or QuoteAsset position
	if node.Symbol.BaseAsset == "USDT" {
		// Если USDT является базовым активом, мы продаем USDT за квотируемый актив
		adjustedQuantity := t.adjustQuantityToLotSize(firstSymbol, currentUSDT)
		initialUSDT = adjustedQuantity
		fmt.Printf("Executing first trade: %s, selling %.2f USDT\n", firstSymbol, adjustedQuantity)
		firstReport, err = t.executor.SellMarket(firstSymbol, adjustedQuantity)
	} else {
		// Если USDT является квотируемым активом, мы покупаем базовый актив за USDT
		// Здесь используем quoteOrderQty - указываем сколько USDT мы хотим потратить
		fmt.Printf("Executing first trade: %s, buying base asset with %.2f USDT\n", firstSymbol, currentUSDT)
		firstReport, err = t.executor.BuyMarketQuote(firstSymbol, currentUSDT)
	}

	if err != nil {
		fmt.Printf("Error executing first trade: %v\n", err)
		return false
	}

	// Update initialUSDT with the actual amount spent from the execution report
	if firstReport.CumQuoteQty > 0 {
		initialUSDT = firstReport.CumQuoteQty
		fmt.Printf("Updated initialUSDT to %.8f USDT based on actual trade execution\n", initialUSDT)
	}

	// Определяем, какой актив мы получили и его количество
	var currentAmount float64
	if firstReport.Side == "1" { // Buy - купили базовый актив
		currentAmount = firstReport.CumQty
		ownerOfCoin = node.Symbol.BaseAsset // Мы получили базовый актив
	} else { // Sell - продали базовый актив, получили квотируемый
		currentAmount = firstReport.CumQuoteQty
		ownerOfCoin = node.Symbol.QuoteAsset // Мы получили квотируемый актив
	}

	fmt.Printf("First trade executed: Got %.8f %s\n", currentAmount, ownerOfCoin)

	// Second trade: Use the asset we got from the first trade
	fmt.Printf("Executing second trade: %s\n", secondNode.SymbolName)
	var secondReport ExecutionReport

	// Determine if we need to buy or sell based on the symbol and what we own
	if secondNode.Symbol.BaseAsset == ownerOfCoin {
		// Мы владеем базовым активом, поэтому продаем его
		fmt.Printf("Selling %.8f %s\n", currentAmount, ownerOfCoin)
		secondReport, err = t.executor.SellMarket(secondNode.SymbolName, currentAmount)
		// После продажи определим, что мы получили, в обработке ниже
	} else {
		// Мы владеем квотируемым активом, поэтому покупаем базовый актив
		// Используем всё количество квотируемого актива, которое у нас есть
		fmt.Printf("Buying base asset with %.8f %s\n", currentAmount, ownerOfCoin)
		secondReport, err = t.executor.BuyMarketQuote(secondNode.SymbolName, currentAmount)
		// После покупки определим, что мы получили, в обработке ниже
	}

	if err != nil {
		fmt.Printf("Error executing second trade: %v\n", err)
		t.executor.Close()
		return false
	}

	// Определяем, какой актив мы получили и его количество
	if secondReport.Side == "1" { // Buy - купили базовый актив
		currentAmount = secondReport.CumQty
		ownerOfCoin = secondNode.Symbol.BaseAsset // Мы получили базовый актив
	} else { // Sell - продали базовый актив, получили квотируемый
		currentAmount = secondReport.CumQuoteQty
		ownerOfCoin = secondNode.Symbol.QuoteAsset // Мы получили квотируемый актив
	}

	fmt.Printf("Second trade executed: Got %.8f %s\n", currentAmount, ownerOfCoin)

	// Third trade: Complete the chain and return to the initial coin (USDT)
	fmt.Printf("Executing third trade: %s\n", lastNode.SymbolName)
	var thirdReport ExecutionReport

	// Determine if we need to buy or sell based on the symbol and what we own
	if lastNode.Symbol.BaseAsset == ownerOfCoin {
		// Мы владеем базовым активом, поэтому продаем его
		fmt.Printf("Selling %.8f %s\n", currentAmount, ownerOfCoin)
		thirdReport, err = t.executor.SellMarket(lastNode.SymbolName, currentAmount)
		// После продажи определим, что мы получили, в обработке ниже
	} else {
		// Мы владеем квотируемым активом, поэтому покупаем базовый актив
		// Используем всё количество квотируемого актива, которое у нас есть
		fmt.Printf("Buying base asset with %.8f %s\n", currentAmount, ownerOfCoin)
		thirdReport, err = t.executor.BuyMarketQuote(lastNode.SymbolName, currentAmount)
		// После покупки определим, что мы получили, в обработке ниже
	}

	if err != nil {
		fmt.Printf("Error executing third trade: %v\n", err)
		t.executor.Close()
		panic(err)
	}

	// Определяем, какой актив мы получили и его количество
	if thirdReport.Side == "1" { // Buy - купили базовый актив
		currentAmount = thirdReport.CumQty
		ownerOfCoin = lastNode.Symbol.BaseAsset // Мы получили базовый актив
	} else { // Sell - продали базовый актив, получили квотируемый
		currentAmount = thirdReport.CumQuoteQty
		ownerOfCoin = lastNode.Symbol.QuoteAsset // Мы получили квотируемый актив
	}

	fmt.Printf("Third trade executed: Got %.8f %s\n", currentAmount, ownerOfCoin)

	// Check if we've returned to the initial coin (USDT)
	if ownerOfCoin == "USDT" {
		fmt.Printf("Successfully completed trading chain, returned to USDT with %.8f\n", currentAmount)
	} else {
		fmt.Printf("Warning: Trading chain did not return to USDT, ended with %.8f %s\n", currentAmount, ownerOfCoin)
	}

	//t.executor.Close()
	//panic("done")
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
			// Use decimal for precise division
			decCurrentAmount := decimal.NewFromFloat(currentAmount)
			decFirstPrice := decimal.NewFromFloat(firstPrice)
			// Check for zero divisor to prevent panic
			if decFirstPrice.IsZero() {
				continue
			}
			decResult := decCurrentAmount.Div(decFirstPrice)
			currentAmount, _ = decResult.Float64()

			// Вторая сделка: определяем направление и какую цену использовать
			var secondPrice float64
			if secondNode.Symbol.BaseAsset == ownerOfCoin {
				// Мы продаем наш ownerOfCoin (например, BTC), используем Bid цену
				secondPrice, ok = t.BidPrice(secondNode.SymbolName)
				ownerOfCoin = secondNode.Symbol.QuoteAsset
				// Use decimal for precise multiplication
				decCurrentAmount := decimal.NewFromFloat(currentAmount)
				decSecondPrice := decimal.NewFromFloat(secondPrice)
				decResult := decCurrentAmount.Mul(decSecondPrice)
				currentAmount, _ = decResult.Float64()
			} else {
				// Мы покупаем базовый актив за наш ownerOfCoin, используем Ask цену
				secondPrice, ok = t.AskPrice(secondNode.SymbolName)
				ownerOfCoin = secondNode.Symbol.BaseAsset
				// Use decimal for precise division
				decCurrentAmount := decimal.NewFromFloat(currentAmount)
				decSecondPrice := decimal.NewFromFloat(secondPrice)
				// Check for zero divisor to prevent panic
				if decSecondPrice.IsZero() {
					continue
				}
				decResult := decCurrentAmount.Div(decSecondPrice)
				currentAmount, _ = decResult.Float64()
			}
			if !ok {
				continue
			}

			// Третья сделка: последний шаг треугольника
			var lastPrice float64
			if lastNode.Symbol.BaseAsset == ownerOfCoin {
				// Мы продаем наш ownerOfCoin, используем Bid цену
				lastPrice, ok = t.BidPrice(lastNode.SymbolName)
				// Use decimal for precise multiplication
				decCurrentAmount := decimal.NewFromFloat(currentAmount)
				decLastPrice := decimal.NewFromFloat(lastPrice)
				decResult := decCurrentAmount.Mul(decLastPrice)
				currentAmount, _ = decResult.Float64()
			} else {
				// Мы покупаем базовый актив за наш ownerOfCoin, используем Ask цену
				lastPrice, ok = t.AskPrice(lastNode.SymbolName)
				// Use decimal for precise division
				decCurrentAmount := decimal.NewFromFloat(currentAmount)
				decLastPrice := decimal.NewFromFloat(lastPrice)
				// Check for zero divisor to prevent panic
				if decLastPrice.IsZero() {
					continue
				}
				decResult := decCurrentAmount.Div(decLastPrice)
				currentAmount, _ = decResult.Float64()
			}
			if !ok {
				continue
			}

			// Use decimal for precise calculation of potential growth
			decAmount := decimal.NewFromFloat(currentAmount)
			decStarted := decimal.NewFromFloat(startedMoney)
			// Check for zero divisor to prevent panic
			if decStarted.IsZero() {
				continue
			}
			decGrowth := decAmount.Div(decStarted).Sub(decimal.NewFromFloat(1)).Mul(decimal.NewFromFloat(100))
			potentialGrowth, _ := decGrowth.Float64()

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
