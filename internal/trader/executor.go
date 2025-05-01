package trader

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/quickfix/config"
	"github.com/quickfixgo/tag"
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
		priv:          privateKey,
		settings:      settings,
		pendingOrders: make(map[string]chan ExecutionReport),
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
	Symbol   string
	Quantity float64
	Side     string // "1" for Buy, "2" for Sell
}

func (e *Executor) BuyMarket(symbol string, quantity float64) (float64, error) {
	params := MarketOrderParams{
		Symbol:   symbol,
		Quantity: quantity,
		Side:     "1", // Buy
	}
	return e.sendMarketOrder(params, "buy")
}

// SellMarket sells the specified symbol at market price
func (e *Executor) SellMarket(symbol string, quantity float64) (float64, error) {
	params := MarketOrderParams{
		Symbol:   symbol,
		Quantity: quantity,
		Side:     "2", // Sell
	}
	return e.sendMarketOrder(params, "sell")
}

// sendMarketOrder creates and sends a market order with the given parameters
func (e *Executor) sendMarketOrder(params MarketOrderParams, orderType string) (float64, error) {
	if e.initiator == nil {
		return 0, fmt.Errorf("FIX initiator not initialized")
	}

	// Create a new message
	msg := quickfix.NewMessage()

	// Set message type to NewOrderSingle (D)
	msg.Header.SetString(tag.MsgType, "D")

	// Generate a unique client order ID
	clOrdID := fmt.Sprintf("order-%d", time.Now().UnixNano())

	// Set required fields for the order
	msg.Body.SetString(tag.ClOrdID, clOrdID)                               // ClOrdID - unique client order ID
	msg.Body.SetString(tag.Symbol, params.Symbol)                          // Symbol
	msg.Body.SetString(tag.Side, params.Side)                              // Side - 1 for Buy, 2 for Sell
	msg.Body.SetString(tag.OrdType, "1")                                   // OrdType - 1 for Market
	msg.Body.SetString(tag.OrderQty, fmt.Sprintf("%.8f", params.Quantity)) // OrderQty as string

	// Create a channel to receive the execution report
	reportChan := make(chan ExecutionReport, 5) // Buffer for multiple reports

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
		e.client.pendingOrdersLock.Lock()
		delete(e.client.pendingOrders, clOrdID)
		e.client.pendingOrdersLock.Unlock()
		close(reportChan)
		return 0, fmt.Errorf("failed to send market %s order: %w", orderType, err)
	}

	// Wait for the execution report with a timeout
	var executionPrice float64
	var lastError error
	timeout := time.After(30 * time.Second)

	for {
		select {
		case report, ok := <-reportChan:
			if !ok {
				// Channel closed, no more reports expected
				if lastError != nil {
					return 0, lastError
				}
				return executionPrice, nil
			}

			// Check if there was an error processing the report
			if report.Error != nil {
				lastError = report.Error
				continue
			}

			// Check if the order was filled
			if report.OrdStatus == "2" { // FILLED
				executionPrice = report.LastPx
				return executionPrice, nil
			}

			// Check if the order was rejected or canceled
			if report.OrdStatus == "8" || report.OrdStatus == "4" { // REJECTED or CANCELED
				return 0, fmt.Errorf("order was %s", report.OrdStatus)
			}

		case <-timeout:
			// Clean up on timeout
			e.client.pendingOrdersLock.Lock()
			delete(e.client.pendingOrders, clOrdID)
			e.client.pendingOrdersLock.Unlock()
			close(reportChan)
			return 0, fmt.Errorf("timeout waiting for execution report")
		}
	}
}

// Close gracefully shuts down the FIX connection
func (e *Executor) Close() {
	if e.initiator != nil {
		e.initiator.Stop()
	}
}

// The ASCII <SOH> that delimits FIX fields.
const soh = "\x01"

// binanceExecutor adapts the QuickFIX/Go “executor” example for Binance **Spot FIX‑OE**.
// See: https://developers.binance.com/docs/binance-spot-api-docs/fix-api
//
// Main differences vs. vanilla executor:
//   * adds Username(553) and a **signed** RawData(96) on Logon[A]
//   * signature is Ed25519 over the concatenated payload required by Binance:
//     "A<SOH>SenderCompID<SOH>TargetCompID<SOH>MsgSeqNum<SOH>SendingTime"
//   * ResetSeqNumFlag(141)=Y on every Logon (Binance wants fresh sequences)

// ExecutionReport holds information about an execution report
type ExecutionReport struct {
	ClOrdID   string
	LastPx    float64
	ExecType  string
	OrdStatus string
	Error     error
}

type binanceExecutor struct {
	priv     ed25519.PrivateKey
	settings *quickfix.Settings // full parsed config, so we can read per‑session params

	// Map to store pending orders and channels to receive execution reports
	pendingOrders     map[string]chan ExecutionReport
	pendingOrdersLock sync.Mutex
}

// --- QuickFIX/Go Application callbacks --------------------------------------------------

func (e *binanceExecutor) OnCreate(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Create %v", sessionID)
}
func (e *binanceExecutor) OnLogon(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Logon %v", sessionID)
}
func (e *binanceExecutor) OnLogout(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Logout %v", sessionID)
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

		// Extract LastPx if this is a trade execution
		if report.ExecType == "F" { // TRADE
			// Get LastPx as a string and convert to float
			if lastPxStr, err := m.Body.GetString(31); err == nil { // 31 is the tag for LastPx
				if lastPx, err := strconv.ParseFloat(lastPxStr, 64); err == nil {
					report.LastPx = lastPx
				} else {
					log.Printf("[FIX] Error parsing LastPx: %v", err)
					report.Error = fmt.Errorf("error parsing LastPx: %w", err)
				}
			} else {
				log.Printf("[FIX] Error extracting LastPx: %v", err)
				report.Error = fmt.Errorf("error extracting LastPx: %w", err)
			}
		}

		// Send the report to the waiting goroutine if there is one
		e.pendingOrdersLock.Lock()
		if ch, ok := e.pendingOrders[report.ClOrdID]; ok {
			ch <- report
			// If the order is terminal (filled, rejected, canceled), remove it from the map
			if report.OrdStatus == "2" || report.OrdStatus == "4" || report.OrdStatus == "8" {
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
