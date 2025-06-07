-- Create database
CREATE DATABASE IF NOT EXISTS alarket;

-- Use the database
USE alarket;

-- Create trades table
CREATE TABLE IF NOT EXISTS trades (
    id String,
    symbol String,
    price Float64,
    quantity Float64,
    buyer_order_id Int64,
    seller_order_id Int64,
    trade_time DateTime64(3),
    is_buyer_maker Bool,
    event_time DateTime64(3),
    created_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(trade_time)
ORDER BY (symbol, trade_time, id)
SETTINGS index_granularity = 8192;

-- Create book_tickers table
CREATE TABLE IF NOT EXISTS book_tickers (
    update_id Int64,
    symbol String,
    best_bid_price Float64,
    best_bid_quantity Float64,
    best_ask_price Float64,
    best_ask_quantity Float64,
    transaction_time DateTime64(3),
    event_time DateTime64(3),
    created_at DateTime64(3) DEFAULT now64(3)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_time)
ORDER BY (symbol, event_time, update_id)
SETTINGS index_granularity = 8192;

-- Indexes are not needed for ClickHouse MergeTree tables
-- Performance is optimized through ORDER BY clause