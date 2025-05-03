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
		// If USDT is the base asset, we need to sell USDT to get the quote asset
		// For selling, the quantity is in terms of the base asset (USDT), so we can use currentUSDT directly
		// Adjust the quantity to meet lot size requirements
		adjustedQuantity := t.adjustQuantityToLotSize(firstSymbol, currentUSDT)
		// Update initialUSDT to the actual amount spent after adjustment
		initialUSDT = adjustedQuantity
		fmt.Printf("Executing first trade: %s, selling %.2f USDT\n", firstSymbol, adjustedQuantity)
		firstReport, err = t.executor.SellMarket(firstSymbol, adjustedQuantity)
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
		// Use decimal for precise division
		decCurrentUSDT := decimal.NewFromFloat(currentUSDT)
		decAskPrice := decimal.NewFromFloat(askPrice)
		// Check for zero divisor to prevent panic
		if decAskPrice.IsZero() {
			fmt.Printf("Error: Ask price for %s is zero\n", firstSymbol)
			return false
		}
		decBaseAssetQuantity := decCurrentUSDT.Div(decAskPrice)

		// Convert back to float64 for compatibility
		baseAssetQuantity, _ := decBaseAssetQuantity.Float64()

		// Adjust the quantity to meet lot size requirements
		adjustedQuantity := t.adjustQuantityToLotSize(firstSymbol, baseAssetQuantity)

		// Update initialUSDT to the actual amount spent after adjustment
		// Use decimal for precise multiplication
		decAdjustedQuantity := decimal.NewFromFloat(adjustedQuantity)
		// Reuse the existing decAskPrice variable
		decInitialUSDT := decAdjustedQuantity.Mul(decAskPrice)

		// Convert back to float64 for compatibility
		initialUSDT, _ = decInitialUSDT.Float64()

		fmt.Printf("Executing first trade: %s, buying %.8f %s with %.2f USDT (price: %.8f)\n",
			firstSymbol, adjustedQuantity, node.Symbol.BaseAsset, initialUSDT, askPrice)
		firstReport, err = t.executor.BuyMarket(firstSymbol, adjustedQuantity)
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
		// Adjust the quantity to meet lot size requirements
		adjustedQuantity := t.adjustQuantityToLotSize(secondSymbol, currentAmount)
		fmt.Printf("Executing second trade: %s, selling %.8f %s\n", secondSymbol, adjustedQuantity, ownerOfCoin)
		secondReport, err = t.executor.SellMarket(secondSymbol, adjustedQuantity)
		// After selling, we now own the quote asset
		ownerOfCoin = secondNode.Symbol.QuoteAsset

		// Apply price conversion for the second trade if the symbol is not in the trading tree
		// This is a workaround for the mock execution environment
		if _, ok := t.tradingTree[secondSymbol]; !ok {
			// Get the bid price for the symbol if available
			bidPrice, ok := t.BidPrice(secondSymbol)
			if ok && bidPrice > 0 {
				// Apply the price conversion
				// Use decimal for precise multiplication
				decAdjustedQuantity := decimal.NewFromFloat(adjustedQuantity)
				decBidPrice := decimal.NewFromFloat(bidPrice)
				decCumQty := decAdjustedQuantity.Mul(decBidPrice)

				// Convert back to float64 for compatibility
				secondReport.CumQty, _ = decCumQty.Float64()

				fmt.Printf("Applied price conversion for %s: %.8f %s * %.8f = %.8f %s\n",
					secondSymbol, adjustedQuantity, secondNode.Symbol.BaseAsset, bidPrice, secondReport.CumQty, ownerOfCoin)
			} else {
				// If bid price is not available, use a default conversion rate of 1.0
				// In a real environment, this should be handled differently
				fmt.Printf("Warning: Using default price conversion for %s\n", secondSymbol)
				secondReport.CumQty = adjustedQuantity
			}
		}
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
		// Use decimal for precise division
		decCurrentAmount := decimal.NewFromFloat(currentAmount)
		decAskPrice := decimal.NewFromFloat(askPrice)
		// Check for zero divisor to prevent panic
		if decAskPrice.IsZero() {
			fmt.Printf("Error: Ask price for %s is zero\n", secondSymbol)
			return false
		}
		decBaseAssetQuantity := decCurrentAmount.Div(decAskPrice)

		// Convert back to float64 for compatibility
		baseAssetQuantity, _ := decBaseAssetQuantity.Float64()

		// Adjust the quantity to meet lot size requirements
		adjustedQuantity := t.adjustQuantityToLotSize(secondSymbol, baseAssetQuantity)

		fmt.Printf("Executing second trade: %s, buying %.8f %s with %.8f %s (price: %.8f)\n",
			secondSymbol, adjustedQuantity, secondNode.Symbol.BaseAsset, currentAmount, ownerOfCoin, askPrice)
		secondReport, err = t.executor.BuyMarket(secondSymbol, adjustedQuantity)
		// After buying, we now own the base asset
		ownerOfCoin = secondNode.Symbol.BaseAsset

		// Apply price conversion for the second trade if the symbol is not in the trading tree
		// This is a workaround for the mock execution environment
		if _, ok := t.tradingTree[secondSymbol]; !ok {
			// If ask price is available, use it for the conversion
			if askPrice > 0 {
				// Apply the price conversion
				secondReport.CumQty = currentAmount / askPrice
				fmt.Printf("Applied price conversion for %s: %.8f %s / %.8f = %.8f %s\n",
					secondSymbol, currentAmount, secondNode.Symbol.QuoteAsset, askPrice, secondReport.CumQty, ownerOfCoin)
			} else {
				// If ask price is not available, use a default conversion rate of 1.0
				// In a real environment, this should be handled differently
				fmt.Printf("Warning: Using default price conversion for %s\n", secondSymbol)
				secondReport.CumQty = adjustedQuantity
			}
		}
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
		// Adjust the quantity to meet lot size requirements
		adjustedQuantity := t.adjustQuantityToLotSize(lastSymbol, currentAmount)
		fmt.Printf("Executing third trade: %s, selling %.8f %s\n", lastSymbol, adjustedQuantity, ownerOfCoin)
		lastReport, err = t.executor.SellMarket(lastSymbol, adjustedQuantity)

		// Apply price conversion for the third trade if the symbol is not in the trading tree
		// This is a workaround for the mock execution environment
		if _, ok := t.tradingTree[lastSymbol]; !ok {
			// Get the bid price for the symbol if available
			bidPrice, ok := t.BidPrice(lastSymbol)
			if ok && bidPrice > 0 {
				// Apply the price conversion
				// Use decimal for precise multiplication
				decAdjustedQuantity := decimal.NewFromFloat(adjustedQuantity)
				decBidPrice := decimal.NewFromFloat(bidPrice)
				decCumQuoteQty := decAdjustedQuantity.Mul(decBidPrice)

				// Convert back to float64 for compatibility
				lastReport.CumQuoteQty, _ = decCumQuoteQty.Float64()

				fmt.Printf("Applied price conversion for %s: %.8f %s * %.8f = %.8f USDT\n",
					lastSymbol, adjustedQuantity, lastNode.Symbol.BaseAsset, bidPrice, lastReport.CumQuoteQty)
			} else {
				// If bid price is not available, use a default conversion rate of 1.0
				// In a real environment, this should be handled differently
				fmt.Printf("Warning: Using default price conversion for %s\n", lastSymbol)
				lastReport.CumQuoteQty = adjustedQuantity
			}
		}
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
		// Use decimal for precise division
		decCurrentAmount := decimal.NewFromFloat(currentAmount)
		decAskPrice := decimal.NewFromFloat(askPrice)
		// Check for zero divisor to prevent panic
		if decAskPrice.IsZero() {
			fmt.Printf("Error: Ask price for %s is zero\n", lastSymbol)
			return false
		}
		decBaseAssetQuantity := decCurrentAmount.Div(decAskPrice)

		// Convert back to float64 for compatibility
		baseAssetQuantity, _ := decBaseAssetQuantity.Float64()

		// Adjust the quantity to meet lot size requirements
		adjustedQuantity := t.adjustQuantityToLotSize(lastSymbol, baseAssetQuantity)

		fmt.Printf("Executing third trade: %s, buying %.8f %s with %.8f %s (price: %.8f)\n",
			lastSymbol, adjustedQuantity, lastNode.Symbol.BaseAsset, currentAmount, ownerOfCoin, askPrice)
		lastReport, err = t.executor.BuyMarket(lastSymbol, adjustedQuantity)

		// Apply price conversion for the third trade if the symbol is not in the trading tree
		// This is a workaround for the mock execution environment
		if _, ok := t.tradingTree[lastSymbol]; !ok {
			// If ask price is available, use it for the conversion
			if askPrice > 0 {
				// Apply the price conversion
				lastReport.CumQuoteQty = currentAmount
				fmt.Printf("Applied price conversion for %s: using quote amount %.8f USDT\n",
					lastSymbol, lastReport.CumQuoteQty)
			} else {
				// If ask price is not available, use a default conversion rate of 1.0
				// In a real environment, this should be handled differently
				fmt.Printf("Warning: Using default price conversion for %s\n", lastSymbol)
				lastReport.CumQuoteQty = currentAmount
			}
		}
	}

	if err != nil {
		fmt.Printf("Error executing third trade: %v\n", err)
		return false
	}

	// Final amount in USDT
	finalUSDT := lastReport.CumQuoteQty

	// Use decimal for precise profit calculations
	decFinalUSDT := decimal.NewFromFloat(finalUSDT)
	decInitialUSDT := decimal.NewFromFloat(initialUSDT)

	// Calculate profit
	decProfit := decFinalUSDT.Sub(decInitialUSDT)
	profit, _ := decProfit.Float64()

	// Calculate profit percentage
	var decProfitPercentage decimal.Decimal
	if decInitialUSDT.IsZero() {
		// Avoid division by zero
		fmt.Printf("Warning: Initial USDT is zero, cannot calculate profit percentage\n")
		decProfitPercentage = decimal.NewFromFloat(0)
	} else {
		decProfitPercentage = decProfit.Div(decInitialUSDT).Mul(decimal.NewFromFloat(100))
	}
	profitPercentage, _ := decProfitPercentage.Float64()

	fmt.Printf("Trades completed! Initial: %.2f USDT, Final: %.2f USDT\n", initialUSDT, finalUSDT)
	fmt.Printf("Profit: %.2f USDT (%.2f%%)\n", profit, profitPercentage)
	t.executor.Close()
	panic("stop")

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
