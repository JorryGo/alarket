package config

import (
	"alarket/internal/binance"
	"alarket/internal/clickhouse"
)

type Config struct {
	Binance    *binance.Config
	ClickHouse *clickhouse.Config
}

func Load() (*Config, error) {
	binanceConfig, err := binance.LoadConfig()
	if err != nil {
		return nil, err
	}

	clickhouseConfig, err := clickhouse.LoadConfig()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Binance:    binanceConfig,
		ClickHouse: clickhouseConfig,
	}

	return cfg, nil
}
