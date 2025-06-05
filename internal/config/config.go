package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Binance    BinanceConfig
	ClickHouse ClickHouseConfig
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

func Load() (*Config, error) {
	// Try to load .env file, but don't fail if it doesn't exist
	_ = godotenv.Load()

	cfg := &Config{}

	// Load Binance configuration
	binanceAPIKey := os.Getenv("BINANCE_API_KEY")
	if binanceAPIKey == "" {
		return nil, fmt.Errorf("BINANCE_API_KEY environment variable is required")
	}

	binanceSecretKey := os.Getenv("BINANCE_SECRET_KEY")
	if binanceSecretKey == "" {
		return nil, fmt.Errorf("BINANCE_SECRET_KEY environment variable is required")
	}

	useTestnet := getEnvOrDefault("BINANCE_USE_TESTNET", true, strconv.ParseBool)

	cfg.Binance = BinanceConfig{
		APIKey:     binanceAPIKey,
		SecretKey:  binanceSecretKey,
		UseTestnet: useTestnet,
	}

	// Load ClickHouse configuration
	cfg.ClickHouse = ClickHouseConfig{
		Host:     getEnvOrDefault("CLICKHOUSE_HOST", "localhost", func(s string) (string, error) { return s, nil }),
		Port:     getEnvOrDefault("CLICKHOUSE_PORT", 9000, strconv.Atoi),
		Database: getEnvOrDefault("CLICKHOUSE_DATABASE", "default", func(s string) (string, error) { return s, nil }),
		Username: getEnvOrDefault("CLICKHOUSE_USERNAME", "default", func(s string) (string, error) { return s, nil }),
		Password: getEnvOrDefault("CLICKHOUSE_PASSWORD", "", func(s string) (string, error) { return s, nil }),
		Debug:    getEnvOrDefault("CLICKHOUSE_DEBUG", false, strconv.ParseBool),
	}

	return cfg, nil
}

func getEnvOrDefault[T any](key string, defaultValue T, parser func(string) (T, error)) T {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := parser(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}
