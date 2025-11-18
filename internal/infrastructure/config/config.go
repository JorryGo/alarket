package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Binance    BinanceConfig
	ClickHouse ClickHouseConfig
	App        AppConfig
}

type BinanceConfig struct {
	APIKey     string
	SecretKey  string
	UseTestnet bool
}

type ClickHouseConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	Debug    bool
}

type AppConfig struct {
	LogLevel             string
	SubscribeTrades      bool
	SubscribeBookTickers bool
	BatchSize            int
	BatchFlushTimeoutMs  int
	Symbols              []string // Specific symbols to collect (empty = all USDT pairs)
}

func Load() (*Config, error) {
	cfg := &Config{}

	// Binance configuration
	cfg.Binance.APIKey = getEnv("BINANCE_API_KEY", "")
	cfg.Binance.SecretKey = getEnv("BINANCE_SECRET_KEY", "")
	cfg.Binance.UseTestnet = getEnvBool("BINANCE_USE_TESTNET", false)

	// ClickHouse configuration
	cfg.ClickHouse.Host = getEnv("CLICKHOUSE_HOST", "localhost")
	cfg.ClickHouse.Port = getEnvInt("CLICKHOUSE_PORT", 9000)
	cfg.ClickHouse.Database = getEnv("CLICKHOUSE_DATABASE", "alarket")
	cfg.ClickHouse.Username = getEnv("CLICKHOUSE_USERNAME", "default")
	cfg.ClickHouse.Password = getEnv("CLICKHOUSE_PASSWORD", "")
	cfg.ClickHouse.Debug = getEnvBool("CLICKHOUSE_DEBUG", false)

	// App configuration
	cfg.App.LogLevel = getEnv("LOG_LEVEL", "info")
	cfg.App.SubscribeTrades = getEnvBool("SUBSCRIBE_TRADES", true)
	cfg.App.SubscribeBookTickers = getEnvBool("SUBSCRIBE_BOOK_TICKERS", false)
	cfg.App.BatchSize = getEnvInt("BATCH_SIZE", 10000)
	cfg.App.BatchFlushTimeoutMs = getEnvInt("BATCH_FLUSH_TIMEOUT_MS", 1000)
	cfg.App.Symbols = getEnvSlice("SYMBOLS", []string{})

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Split by comma and trim spaces
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}
