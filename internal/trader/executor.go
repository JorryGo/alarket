package trader

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/quickfix/config"
)

//const apiKey = "ntCyqh0qiuEGtg9jH4EOtIoeSv8ETOjiHdeHjs0zNgwpckL5flXpigOIT5teYmzv"
//const secretKey = "0x26WvMtZietFtv5nGvqQnM2WrIlXTSzQGiCWej045lus55OKo5bemF1HHPQ3Vtn"

type Executor struct {
	client *binanceExecutor
}

func InitExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) BuyMarket(symbol string, quantity float64) (float64, error) {
	return 1.4, nil
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

type binanceExecutor struct {
	priv     ed25519.PrivateKey
	settings *quickfix.Settings // full parsed config, so we can read per‑session params
}

// --- QuickFIX/Go Application callbacks --------------------------------------------------

func (e *binanceExecutor) OnCreate(sessionID quickfix.SessionID) {}
func (e *binanceExecutor) OnLogon(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Logon %v", sessionID)
}
func (e *binanceExecutor) OnLogout(sessionID quickfix.SessionID) {
	log.Printf("[FIX] Logout %v", sessionID)
}
func (e *binanceExecutor) FromAdmin(msg *quickfix.Message, _ quickfix.SessionID) error { return nil }
func (e *binanceExecutor) ToApp(_ *quickfix.Message, _ quickfix.SessionID) error       { return nil }
func (e *binanceExecutor) FromApp(m *quickfix.Message, _ quickfix.SessionID) quickfix.MessageRejectError {
	msgType, _ := m.MsgType()
	log.Printf("[FIX] inbound %s: %s", msgType, m.String())
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

	if sessCfg, ok := e.settings.SessionSettings()[sid]; ok {
		user, _ = sessCfg.Setting("Username")
		hbInt, _ = sessCfg.IntSetting(config.HeartBtInt)
	}
	if user == "" { // fall back to [DEFAULT]
		user, _ = e.settings.GlobalSettings().Setting("Username")
	}
	if hbInt == 0 {
		hbInt, _ = e.settings.GlobalSettings().IntSetting(config.HeartBtInt)
	}

	// Minimal mandatory body fields.
	m.Body.SetString(553, user) // Username – Binance uses this for API Key
	m.Body.SetInt(98, 0)        // EncryptMethod=0 (none)
	m.Body.SetInt(108, hbInt)   // HeartBtInt seconds
	m.Body.SetBool(141, true)   // ResetSeqNumFlag=Y

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
