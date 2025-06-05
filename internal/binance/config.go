package binance

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	APIKey     string
	SecretKey  string
	UseTestnet bool
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	binanceAPIKey := os.Getenv("BINANCE_API_KEY")
	if binanceAPIKey == "" {
		return nil, fmt.Errorf("BINANCE_API_KEY environment variable is required")
	}

	binanceSecretKey := os.Getenv("BINANCE_SECRET_KEY")
	if binanceSecretKey == "" {
		return nil, fmt.Errorf("BINANCE_SECRET_KEY environment variable is required")
	}

	useTestnet := getEnvOrDefault("BINANCE_USE_TESTNET", true, strconv.ParseBool)

	cfg := &Config{
		APIKey:     binanceAPIKey,
		SecretKey:  binanceSecretKey,
		UseTestnet: useTestnet,
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
