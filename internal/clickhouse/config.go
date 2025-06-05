package clickhouse

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	Debug    bool
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
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