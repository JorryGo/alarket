package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear environment variables
	clearEnvVars()
	
	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	
	// Test Binance defaults
	assert.Equal(t, "", cfg.Binance.APIKey)
	assert.Equal(t, "", cfg.Binance.SecretKey)
	assert.False(t, cfg.Binance.UseTestnet)
	
	// Test ClickHouse defaults
	assert.Equal(t, "localhost", cfg.ClickHouse.Host)
	assert.Equal(t, 9000, cfg.ClickHouse.Port)
	assert.Equal(t, "alarket", cfg.ClickHouse.Database)
	assert.Equal(t, "default", cfg.ClickHouse.Username)
	assert.Equal(t, "", cfg.ClickHouse.Password)
	assert.False(t, cfg.ClickHouse.Debug)
	
	// Test App defaults
	assert.Equal(t, "info", cfg.App.LogLevel)
	assert.True(t, cfg.App.SubscribeTrades)
	assert.False(t, cfg.App.SubscribeBookTickers)
	assert.Equal(t, 10000, cfg.App.BatchSize)
	assert.Equal(t, 1000, cfg.App.BatchFlushTimeoutMs)
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// Clear environment variables first
	clearEnvVars()
	
	// Set test environment variables
	testEnvVars := map[string]string{
		"BINANCE_API_KEY":           "test_api_key",
		"BINANCE_SECRET_KEY":        "test_secret_key",
		"BINANCE_USE_TESTNET":       "true",
		"CLICKHOUSE_HOST":           "test.clickhouse.com",
		"CLICKHOUSE_PORT":           "8123",
		"CLICKHOUSE_DATABASE":       "test_db",
		"CLICKHOUSE_USERNAME":       "test_user",
		"CLICKHOUSE_PASSWORD":       "test_password",
		"CLICKHOUSE_DEBUG":          "true",
		"LOG_LEVEL":                 "debug",
		"SUBSCRIBE_TRADES":          "false",
		"SUBSCRIBE_BOOK_TICKERS":    "true",
		"BATCH_SIZE":                "5000",
		"BATCH_FLUSH_TIMEOUT_MS":    "500",
	}
	
	for key, value := range testEnvVars {
		os.Setenv(key, value)
	}
	defer clearEnvVars()
	
	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	
	// Test Binance configuration
	assert.Equal(t, "test_api_key", cfg.Binance.APIKey)
	assert.Equal(t, "test_secret_key", cfg.Binance.SecretKey)
	assert.True(t, cfg.Binance.UseTestnet)
	
	// Test ClickHouse configuration
	assert.Equal(t, "test.clickhouse.com", cfg.ClickHouse.Host)
	assert.Equal(t, 8123, cfg.ClickHouse.Port)
	assert.Equal(t, "test_db", cfg.ClickHouse.Database)
	assert.Equal(t, "test_user", cfg.ClickHouse.Username)
	assert.Equal(t, "test_password", cfg.ClickHouse.Password)
	assert.True(t, cfg.ClickHouse.Debug)
	
	// Test App configuration
	assert.Equal(t, "debug", cfg.App.LogLevel)
	assert.False(t, cfg.App.SubscribeTrades)
	assert.True(t, cfg.App.SubscribeBookTickers)
	assert.Equal(t, 5000, cfg.App.BatchSize)
	assert.Equal(t, 500, cfg.App.BatchFlushTimeoutMs)
}

func TestGetEnv(t *testing.T) {
	t.Run("existing environment variable", func(t *testing.T) {
		os.Setenv("TEST_KEY", "test_value")
		defer os.Unsetenv("TEST_KEY")
		
		value := getEnv("TEST_KEY", "default")
		assert.Equal(t, "test_value", value)
	})
	
	t.Run("non-existing environment variable", func(t *testing.T) {
		value := getEnv("NON_EXISTING_KEY", "default")
		assert.Equal(t, "default", value)
	})
	
	t.Run("empty environment variable", func(t *testing.T) {
		os.Setenv("EMPTY_KEY", "")
		defer os.Unsetenv("EMPTY_KEY")
		
		value := getEnv("EMPTY_KEY", "default")
		assert.Equal(t, "default", value)
	})
}

func TestGetEnvInt(t *testing.T) {
	t.Run("valid integer environment variable", func(t *testing.T) {
		os.Setenv("TEST_INT", "42")
		defer os.Unsetenv("TEST_INT")
		
		value := getEnvInt("TEST_INT", 100)
		assert.Equal(t, 42, value)
	})
	
	t.Run("invalid integer environment variable", func(t *testing.T) {
		os.Setenv("TEST_INVALID_INT", "not_a_number")
		defer os.Unsetenv("TEST_INVALID_INT")
		
		value := getEnvInt("TEST_INVALID_INT", 100)
		assert.Equal(t, 100, value)
	})
	
	t.Run("non-existing integer environment variable", func(t *testing.T) {
		value := getEnvInt("NON_EXISTING_INT", 100)
		assert.Equal(t, 100, value)
	})
	
	t.Run("empty integer environment variable", func(t *testing.T) {
		os.Setenv("EMPTY_INT", "")
		defer os.Unsetenv("EMPTY_INT")
		
		value := getEnvInt("EMPTY_INT", 100)
		assert.Equal(t, 100, value)
	})
	
	t.Run("negative integer", func(t *testing.T) {
		os.Setenv("TEST_NEGATIVE_INT", "-42")
		defer os.Unsetenv("TEST_NEGATIVE_INT")
		
		value := getEnvInt("TEST_NEGATIVE_INT", 100)
		assert.Equal(t, -42, value)
	})
	
	t.Run("zero integer", func(t *testing.T) {
		os.Setenv("TEST_ZERO_INT", "0")
		defer os.Unsetenv("TEST_ZERO_INT")
		
		value := getEnvInt("TEST_ZERO_INT", 100)
		assert.Equal(t, 0, value)
	})
}

func TestGetEnvBool(t *testing.T) {
	testCases := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"true string", "true", true},
		{"false string", "false", false},
		{"1 as true", "1", true},
		{"0 as false", "0", false},
		{"True with capital", "True", true},
		{"False with capital", "False", false},
		{"TRUE all caps", "TRUE", true},
		{"FALSE all caps", "FALSE", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("TEST_BOOL", tc.envValue)
			defer os.Unsetenv("TEST_BOOL")
			
			value := getEnvBool("TEST_BOOL", false)
			assert.Equal(t, tc.expected, value)
		})
	}
	
	t.Run("invalid boolean environment variable", func(t *testing.T) {
		os.Setenv("TEST_INVALID_BOOL", "not_a_bool")
		defer os.Unsetenv("TEST_INVALID_BOOL")
		
		value := getEnvBool("TEST_INVALID_BOOL", true)
		assert.True(t, value) // Should return default
	})
	
	t.Run("non-existing boolean environment variable", func(t *testing.T) {
		value := getEnvBool("NON_EXISTING_BOOL", true)
		assert.True(t, value) // Should return default
	})
	
	t.Run("empty boolean environment variable", func(t *testing.T) {
		os.Setenv("EMPTY_BOOL", "")
		defer os.Unsetenv("EMPTY_BOOL")
		
		value := getEnvBool("EMPTY_BOOL", true)
		assert.True(t, value) // Should return default
	})
}

func TestLoad_PartialEnvironmentVariables(t *testing.T) {
	// Clear environment variables
	clearEnvVars()
	
	// Set only some environment variables
	os.Setenv("BINANCE_API_KEY", "partial_api_key")
	os.Setenv("CLICKHOUSE_PORT", "8000")
	os.Setenv("LOG_LEVEL", "warn")
	defer clearEnvVars()
	
	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	
	// Test overridden values
	assert.Equal(t, "partial_api_key", cfg.Binance.APIKey)
	assert.Equal(t, 8000, cfg.ClickHouse.Port)
	assert.Equal(t, "warn", cfg.App.LogLevel)
	
	// Test default values for non-set variables
	assert.Equal(t, "", cfg.Binance.SecretKey)
	assert.Equal(t, "localhost", cfg.ClickHouse.Host)
	assert.True(t, cfg.App.SubscribeTrades)
}

// Helper function to clear all environment variables used in config
func clearEnvVars() {
	envVars := []string{
		"BINANCE_API_KEY",
		"BINANCE_SECRET_KEY", 
		"BINANCE_USE_TESTNET",
		"CLICKHOUSE_HOST",
		"CLICKHOUSE_PORT",
		"CLICKHOUSE_DATABASE",
		"CLICKHOUSE_USERNAME",
		"CLICKHOUSE_PASSWORD",
		"CLICKHOUSE_DEBUG",
		"LOG_LEVEL",
		"SUBSCRIBE_TRADES",
		"SUBSCRIBE_BOOK_TICKERS",
		"BATCH_SIZE",
		"BATCH_FLUSH_TIMEOUT_MS",
	}
	
	for _, key := range envVars {
		os.Unsetenv(key)
	}
}