package trader

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/quickfix/config"
	"github.com/quickfixgo/tag"
	"github.com/shopspring/decimal"
)

// Константы для конфигурации
const (
	DEFAULT_RETRY_COUNT = 3 // Дефолтное количество повторных попыток (всего будет 3 попытки: 1 начальная + 2 повторных)
)

//const apiKey = "ntCyqh0qiuEGtg9jH4EOtIoeSv8ETOjiHdeHjs0zNgwpckL5flXpigOIT5teYmzv"
//const secretKey = "0x26WvMtZietFtv5nGvqQnM2WrIlXTSzQGiCWej045lus55OKo5bemF1HHPQ3Vtn"

type Executor struct {
	client    *binanceExecutor
	initiator *quickfix.Initiator
}

func InitExecutor() *Executor {
	// Open settings file
	settingsFile, err := os.Open("FIXAssets\\settings.ini")
	if err != nil {
		log.Printf("[FIX] Error opening settings file: %v", err)
		return &Executor{}
	}
	defer settingsFile.Close()

	// Parse settings
	settings, err := quickfix.ParseSettings(settingsFile)
	if err != nil {
		log.Printf("[FIX] Error parsing settings: %v", err)
		return &Executor{}
	}

	// Load private key
	privateKey, err := loadEd25519Key("FIXAssets\\pkey.pem")
	if err != nil {
		log.Printf("[FIX] Error loading private key: %v", err)
		return &Executor{}
	}

	// Initialize binanceExecutor
	client := &binanceExecutor{
		priv:            privateKey,
		settings:        settings,
		pendingOrders:   make(map[string]chan ExecutionReport),
		isConnected:     false,
		isAuthenticated: false,
	}

	// Create initiator
	initiator, err := quickfix.NewInitiator(client, quickfix.NewMemoryStoreFactory(), settings, quickfix.NewScreenLogFactory())
	if err != nil {
		log.Printf("[FIX] Error creating initiator: %v", err)
		return &Executor{
			client: client,
		}
	}

	// Start initiator to connect immediately
	err = initiator.Start()
	if err != nil {
		log.Printf("[FIX] Error starting initiator: %v", err)
		return &Executor{
			client: client,
		}
	}

	return &Executor{
		client:    client,
		initiator: initiator,
	}
}

// MarketOrderParams holds parameters for a market order
type MarketOrderParams struct {
	Symbol          string
	Quantity        float64
	Side            string // "1" for Buy, "2" for Sell
	IsQuoteOrderQty bool
	RetryCount      int     // Количество оставшихся попыток повторения
	StepSize        float64 // Step size for the trading pair
}

// BuyMarket is now a method that can be exposed but is mainly for backwards compatibility
// It directly calls SendMarketOrder with default stepSize
func (e *Executor) BuyMarket(symbol string, quantity float64) (ExecutionReport, error) {
	params := MarketOrderParams{
		Symbol:          symbol,
		Quantity:        quantity,
		Side:            "1",   // Buy
		IsQuoteOrderQty: false, // Обычный ордер на количество базового актива
		RetryCount:      DEFAULT_RETRY_COUNT,
		StepSize:        0, // Default stepSize
	}
	return e.SendMarketOrder(params, "buy")
}

// SellMarket is now a method that can be exposed but is mainly for backwards compatibility
// It directly calls SendMarketOrder with default stepSize
func (e *Executor) SellMarket(symbol string, quantity float64) (ExecutionReport, error) {
	params := MarketOrderParams{
		Symbol:          symbol,
		Quantity:        quantity,
		Side:            "2",   // Sell
		IsQuoteOrderQty: false, // Обычный ордер на количество базового актива
		RetryCount:      DEFAULT_RETRY_COUNT,
		StepSize:        0, // Default stepSize
	}
	return e.SendMarketOrder(params, "sell")
}

// BuyMarketQuote is now a method that can be exposed but is mainly for backwards compatibility
// It directly calls SendMarketOrder with default stepSize
func (e *Executor) BuyMarketQuote(symbol string, quoteQuantity float64) (ExecutionReport, error) {
	params := MarketOrderParams{
		Symbol:          symbol,
		Quantity:        quoteQuantity,
		Side:            "1",  // Buy
		IsQuoteOrderQty: true, // Ордер на сумму квотируемого актива
		RetryCount:      DEFAULT_RETRY_COUNT,
		StepSize:        0, // Default stepSize
	}
	return e.SendMarketOrder(params, "buy")
}

// SellMarketQuote is now a method that can be exposed but is mainly for backwards compatibility
// It directly calls SendMarketOrder with default stepSize
func (e *Executor) SellMarketQuote(symbol string, quoteQuantity float64) (ExecutionReport, error) {
	params := MarketOrderParams{
		Symbol:          symbol,
		Quantity:        quoteQuantity,
		Side:            "2",  // Sell
		IsQuoteOrderQty: true, // Ордер на сумму квотируемого актива
		RetryCount:      DEFAULT_RETRY_COUNT,
		StepSize:        0, // Default stepSize
	}
	return e.SendMarketOrder(params, "sell")
}

// safeClose безопасно закрывает канал, игнорируя панику, если канал уже закрыт
func safeClose(ch chan ExecutionReport) {
	defer func() {
		if r := recover(); r != nil {
			// Игнорируем панику при закрытии канала, канал уже закрыт
			log.Printf("[FIX] Warning: Attempted to close already closed channel: %v", r)
		}
	}()
	close(ch)
}

// SendMarketOrder creates and sends a market order with the given parameters
func (e *Executor) SendMarketOrder(params MarketOrderParams, orderType string) (ExecutionReport, error) {
	// Create an empty report
	emptyReport := ExecutionReport{}

	if e.initiator == nil {
		return emptyReport, fmt.Errorf("FIX initiator not initialized")
	}

	// Check if the connection is active and authenticated
	if !e.IsConnected() {
		return emptyReport, fmt.Errorf("FIX connection is not active or not authenticated")
	}

	// Create a new message
	msg := quickfix.NewMessage()

	// Set message type to NewOrderSingle (D)
	msg.Header.SetString(tag.MsgType, "D")

	// Generate a unique client order ID
	clOrdID := fmt.Sprintf("order-%d", time.Now().UnixNano())

	// Set required fields for the order
	msg.Body.SetString(tag.ClOrdID, clOrdID)      // ClOrdID - unique client order ID
	msg.Body.SetString(tag.Symbol, params.Symbol) // Symbol
	msg.Body.SetString(tag.Side, params.Side)     // Side - 1 for Buy, 2 for Sell
	msg.Body.SetString(tag.OrdType, "1")          // OrdType - 1 for Market

	// Prepare quantity string based on step size
	var quantityStr string

	// Check if stepSize is 1 (or close to 1), which indicates whole number requirement
	if params.StepSize >= 0.99 && params.StepSize <= 1.01 {
		// Format as integer (whole number)
		qty := math.Floor(params.Quantity)              // Ensure it's a whole number
		quantityStr = strconv.FormatInt(int64(qty), 10) // Format as integer without decimal places
		log.Printf("[FIX] Using integer format for quantity: %s (stepSize=%f)", quantityStr, params.StepSize)
	} else {
		// Use decimal format with precision based on step size
		decQuantity := decimal.NewFromFloat(params.Quantity)
		// Format with 8 decimal places but trim trailing zeros
		quantityStr = decQuantity.StringFixed(8)
		log.Printf("[FIX] Using decimal format for quantity: %s", quantityStr)
	}

	// В зависимости от типа ордера, устанавливаем либо OrderQty, либо QuoteOrderQty
	if params.IsQuoteOrderQty {
		// Используем тег 152 (CashOrderQty) для указания суммы в квотируемом активе
		msg.Body.SetString(tag.CashOrderQty, quantityStr) // CashOrderQty как строка
	} else {
		// Обычный ордер на определенное количество базового актива
		msg.Body.SetString(tag.OrderQty, quantityStr) // OrderQty как строка
	}

	// Create a channel to receive the execution report
	reportChan := make(chan ExecutionReport, 5) // Buffer for multiple reports

	// Определяем функцию очистки ресурсов с безопасным закрытием канала
	cleanupResources := func() {
		e.client.pendingOrdersLock.Lock()
		delete(e.client.pendingOrders, clOrdID)
		e.client.pendingOrdersLock.Unlock()
		safeClose(reportChan)
	}

	// Register the order in the pending orders map
	e.client.pendingOrdersLock.Lock()
	e.client.pendingOrders[clOrdID] = reportChan
	e.client.pendingOrdersLock.Unlock()

	// Create a session ID based on the settings.ini file
	sessionID := quickfix.SessionID{
		BeginString:  "FIX.4.4",
		SenderCompID: "MYBOT1",
		TargetCompID: "SPOT",
	}

	// Send the order
	err := quickfix.SendToTarget(msg, sessionID)
	if err != nil {
		// Clean up if sending fails
		cleanupResources()
		return emptyReport, fmt.Errorf("failed to send market %s order: %w", orderType, err)
	}

	// Pre-populate some fields in the report that we know from the order
	emptyReport.ClOrdID = clOrdID
	emptyReport.Symbol = params.Symbol
	emptyReport.Side = params.Side
	emptyReport.OrderQty = params.Quantity

	// Wait for the execution report with a timeout
	var lastError error
	timeout := time.After(30 * time.Second)

	for {
		select {
		case report, ok := <-reportChan:
			if !ok {
				// Channel closed, no more reports expected
				if lastError != nil {
					return emptyReport, lastError
				}
				// This should not happen normally, but return what we have
				return emptyReport, nil
			}

			// Check if there was an error processing the report
			if report.Error != nil {
				lastError = report.Error
				continue
			}

			// Check if the order was filled
			if report.OrdStatus == "2" { // FILLED
				return report, nil
			}

			// Check if the order was rejected
			if report.OrdStatus == "8" { // REJECTED
				// Если еще остались попытки, выполняем рекурсивный вызов
				if params.RetryCount > 0 {
					// Уменьшаем счетчик оставшихся попыток
					params.RetryCount--

					// Выводим информацию о повторной попытке
					retryNumber := DEFAULT_RETRY_COUNT - params.RetryCount
					totalAttempts := DEFAULT_RETRY_COUNT + 1 // +1 потому что начальная попытка тоже считается
					fmt.Printf("Order %s rejected (status 8), retrying attempt %d/%d...\n",
						clOrdID, retryNumber, totalAttempts)

					// Очищаем ресурсы текущей попытки
					cleanupResources()

					// Ждем немного перед повторной попыткой
					time.Sleep(500 * time.Millisecond)

					// Рекурсивно вызываем SendMarketOrder с обновленными параметрами
					return e.SendMarketOrder(params, orderType)
				}

				// Если попытки закончились, возвращаем ошибку
				return report, fmt.Errorf("order was %s after multiple retries", report.OrdStatus)
			}

			// Check if order was canceled
			if report.OrdStatus == "4" { // CANCELED
				return report, fmt.Errorf("order was %s", report.OrdStatus)
			}

			return report, fmt.Errorf("unknown condition")

		case <-timeout:
			// Clean up on timeout
			cleanupResources()
			return emptyReport, fmt.Errorf("timed out waiting for %s order execution report", orderType)
		}
	}
}

// IsConnected returns true if the FIX connection is active and authenticated
func (e *Executor) IsConnected() bool {
	if e.client == nil {
		return false
	}
	return e.client.isConnectedAndAuthenticated()
}

// Close gracefully shuts down the FIX connection
func (e *Executor) Close() {
	if e.initiator != nil {
		e.initiator.Stop()
	}
}

// isConnectedAndAuthenticated returns true if the connection is active and authenticated
func (e *binanceExecutor) isConnectedAndAuthenticated() bool {
	e.statusLock.RLock()
	defer e.statusLock.RUnlock()
	return e.isConnected && e.isAuthenticated
}

// The ASCII <SOH> that delimits FIX fields.
const soh = "\x01"

// binanceExecutor adapts the QuickFIX/Go "executor" example for Binance **Spot FIX-OE**.
// See: https://developers.binance.com/docs/binance-spot-api-docs/fix-api
//
// Main differences vs. vanilla executor:
//   * adds Username(553) and a **signed** RawData(96) on Logon[A]
//   * signature is Ed25519 over the concatenated payload required by Binance:
//     "A<SOH>SenderCompID<SOH>TargetCompID<SOH>MsgSeqNum<SOH>SendingTime"
//   * ResetSeqNumFlag(141)=Y on every Logon (Binance wants fresh sequences)

// ExecutionReport holds information about an execution report
type ExecutionReport struct {
	ClOrdID     string  // Client order ID
	OrderID     string  // Exchange order ID
	Symbol      string  // Trading symbol
	Side        string  // "1" for Buy, "2" for Sell
	OrderQty    float64 // Requested quantity
	LastQty     float64 // Executed quantity in the last fill
	CumQty      float64 // Cumulative executed quantity
	LastPx      float64 // Last execution price
	CumQuoteQty float64 // Cumulative executed quantity in quote currency
	ExecType    string  // Execution type
	OrdStatus   string  // Order status
	Error       error   // Any error that occurred
}

type binanceExecutor struct {
	priv     ed25519.PrivateKey
	settings *quickfix.Settings // full parsed config, so we can read per-session params

	// Map to store pending orders and channels to receive execution reports
	pendingOrders     map[string]chan ExecutionReport
	pendingOrdersLock sync.Mutex

	// Connection status
	isConnected     bool
	isAuthenticated bool
	statusLock      sync.RWMutex
}

// --- QuickFIX/Go Application callbacks --------------------------------------------------

func (e *binanceExecutor) OnCreate(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Create %v", sessionID)
	e.statusLock.Lock()
	e.isConnected = true
	e.statusLock.Unlock()
}
func (e *binanceExecutor) OnLogon(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Logon %v", sessionID)
	e.statusLock.Lock()
	e.isConnected = true
	e.isAuthenticated = true
	e.statusLock.Unlock()
}
func (e *binanceExecutor) OnLogout(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Logout %v", sessionID)
	e.statusLock.Lock()
	e.isConnected = false
	e.isAuthenticated = false
	e.statusLock.Unlock()
}
func (e *binanceExecutor) FromAdmin(msg *quickfix.Message, _ quickfix.SessionID) quickfix.MessageRejectError {
	msgType, _ := msg.MsgType()
	log.Printf("[FIX] inbound %s: %s", msgType, msg.String())
	return nil
}
func (e *binanceExecutor) ToApp(_ *quickfix.Message, _ quickfix.SessionID) error { return nil }
func (e *binanceExecutor) FromApp(m *quickfix.Message, _ quickfix.SessionID) quickfix.MessageRejectError {
	msgType, _ := m.MsgType()
	log.Printf("[FIX] inbound %s: %s", msgType, m.String())

	// Check if this is an execution report
	if msgType == "8" { // ExecutionReport
		var report ExecutionReport

		// Extract ClOrdID
		if clOrdID, err := m.Body.GetString(11); err == nil { // 11 is the tag for ClOrdID
			report.ClOrdID = clOrdID
		} else {
			log.Printf("[FIX] Error extracting ClOrdID: %v", err)
			return nil
		}

		// Extract OrderID
		if orderID, err := m.Body.GetString(37); err == nil { // 37 is the tag for OrderID
			report.OrderID = orderID
		} else {
			log.Printf("[FIX] Warning: Error extracting OrderID: %v", err)
			// Not returning, as this field might be missing in some reports
		}

		// Extract Symbol
		if symbol, err := m.Body.GetString(55); err == nil { // 55 is the tag for Symbol
			report.Symbol = symbol
		} else {
			log.Printf("[FIX] Warning: Error extracting Symbol: %v", err)
			// Not returning, as this field might be missing in some reports
		}

		// Extract Side
		if side, err := m.Body.GetString(54); err == nil { // 54 is the tag for Side
			report.Side = side
		} else {
			log.Printf("[FIX] Warning: Error extracting Side: %v", err)
			// Not returning, as this field might be missing in some reports
		}

		// Extract ExecType
		if execType, err := m.Body.GetString(150); err == nil { // 150 is the tag for ExecType
			report.ExecType = execType
		} else {
			log.Printf("[FIX] Error extracting ExecType: %v", err)
			return nil
		}

		// Extract OrdStatus
		if ordStatus, err := m.Body.GetString(39); err == nil { // 39 is the tag for OrdStatus
			report.OrdStatus = ordStatus
		} else {
			log.Printf("[FIX] Error extracting OrdStatus: %v", err)
			return nil
		}

		// Extract OrderQty
		if orderQtyStr, err := m.Body.GetString(38); err == nil { // 38 is the tag for OrderQty
			if orderQty, err := strconv.ParseFloat(orderQtyStr, 64); err == nil {
				report.OrderQty = orderQty
			} else {
				log.Printf("[FIX] Warning: Error parsing OrderQty: %v", err)
			}
		} else {
			log.Printf("[FIX] Warning: Error extracting OrderQty: %v", err)
		}

		// Extract LastQty
		if lastQtyStr, err := m.Body.GetString(32); err == nil { // 32 is the tag for LastQty
			if lastQty, err := strconv.ParseFloat(lastQtyStr, 64); err == nil {
				report.LastQty = lastQty
			} else {
				log.Printf("[FIX] Warning: Error parsing LastQty: %v", err)
			}
		} else {
			log.Printf("[FIX] Warning: Error extracting LastQty: %v", err)
		}

		// Extract CumQty
		if cumQtyStr, err := m.Body.GetString(14); err == nil { // 14 is the tag for CumQty
			if cumQty, err := strconv.ParseFloat(cumQtyStr, 64); err == nil {
				report.CumQty = cumQty
			} else {
				log.Printf("[FIX] Warning: Error parsing CumQty: %v", err)
			}
		} else {
			log.Printf("[FIX] Warning: Error extracting CumQty: %v", err)
		}

		// Extract LastPx
		if lastPxStr, err := m.Body.GetString(31); err == nil { // 31 is the tag for LastPx
			if lastPx, err := strconv.ParseFloat(lastPxStr, 64); err == nil {
				report.LastPx = lastPx
			} else {
				log.Printf("[FIX] Warning: Error parsing LastPx: %v", err)
				report.Error = fmt.Errorf("error parsing LastPx: %w", err)
			}
		} else {
			log.Printf("[FIX] Warning: Error extracting LastPx: %v", err)
		}

		// Extract CumQuoteQty
		if cumQuoteQtyStr, err := m.Body.GetString(25017); err == nil { // 25017 is the tag for CumQuoteQty
			if cumQuoteQty, err := strconv.ParseFloat(cumQuoteQtyStr, 64); err == nil {
				report.CumQuoteQty = cumQuoteQty
			} else {
				log.Printf("[FIX] Warning: Error parsing CumQuoteQty: %v", err)
			}
		} else {
			log.Printf("[FIX] Warning: Error extracting CumQuoteQty: %v", err)
		}

		// Send the report to the waiting goroutine if there is one
		e.pendingOrdersLock.Lock()
		if ch, ok := e.pendingOrders[report.ClOrdID]; ok {
			// If the order is terminal (filled, rejected, canceled), remove it from the map
			if report.OrdStatus == "2" || report.OrdStatus == "4" || report.OrdStatus == "8" {
				ch <- report
				close(ch)
				delete(e.pendingOrders, report.ClOrdID)
			}
		}
		e.pendingOrdersLock.Unlock()
	}

	return nil
}

// ToAdmin lets us tweak the outgoing Logon <A>.
func (e *binanceExecutor) ToAdmin(m *quickfix.Message, sid quickfix.SessionID) {
	msgType, _ := m.MsgType()
	if msgType != "A" { // only intercept Logon
		return
	}

	// Pull session/global settings.
	var hbInt int
	var user string
	var messageHandling int

	if sessCfg, ok := e.settings.SessionSettings()[sid]; ok {
		user, _ = sessCfg.Setting("Username")
		hbInt, _ = sessCfg.IntSetting(config.HeartBtInt)
		messageHandling, _ = sessCfg.IntSetting("MessageHandling")
	}
	if user == "" { // fall back to [DEFAULT]
		user, _ = e.settings.GlobalSettings().Setting("Username")
	}
	if hbInt == 0 {
		hbInt, _ = e.settings.GlobalSettings().IntSetting(config.HeartBtInt)
	}
	if messageHandling == 0 {
		messageHandling, _ = e.settings.GlobalSettings().IntSetting("MessageHandling")
	}

	// Minimal mandatory body fields.
	m.Body.SetString(553, user)           // Username – Binance uses this for API Key
	m.Body.SetInt(98, 0)                  // EncryptMethod=0 (none)
	m.Body.SetInt(108, hbInt)             // HeartBtInt seconds
	m.Body.SetBool(141, true)             // ResetSeqNumFlag=Y
	m.Body.SetInt(25035, messageHandling) // MessageHandling

	// Build Binance-specific payload to sign.
	seqNum, _ := m.Header.GetInt(34)
	sendingTime, _ := m.Header.GetString(52)
	payload := strings.Join([]string{
		"A",                // MsgType
		sid.SenderCompID,   // 49
		sid.TargetCompID,   // 56 ("SPOT")
		fmt.Sprint(seqNum), // 34
		sendingTime,        // 52
	}, soh)

	sig := ed25519.Sign(e.priv, []byte(payload))
	b64 := base64.StdEncoding.EncodeToString(sig)

	m.Body.SetString(96, b64)   // RawData
	m.Body.SetInt(95, len(b64)) // RawDataLength
}

// --- Helpers ---------------------------------------------------------------------------

// loadEd25519Key expects an **unencrypted** PKCS8 Ed25519 PEM.
func loadEd25519Key(path string) (ed25519.PrivateKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", path)
	}
	pk, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	edKey, ok := pk.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not Ed25519 private key")
	}
	return edKey, nil
}
