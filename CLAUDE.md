# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Requirements

**Always place compiled binaries in `./build`, never in the repo root.  
Every `go build` command MUST use `-o ./build/<name>` and create the folder if it doesn't exist.**

## Build Commands

Build the trade collector:
```bash
mkdir -p ./build && go build -o ./build/trade-collector cmd/trade-collector/main.go
```

Run the application:
```bash
./build/trade-collector
```

## Project Architecture

This is a Go-based cryptocurrency trading data collector that streams real-time trade data from Binance WebSocket API and stores it in ClickHouse database.

### Core Components

- **cmd/trade-collector/main.go**: Main application entry point that coordinates the entire system
- **internal/connector**: WebSocket connection pool manager that handles multiple concurrent connections to Binance streams
- **internal/binance**: Binance API integration layer with event handling and ticker processing
- **internal/clickhouse**: ClickHouse database service for data persistence
- **internal/config**: Configuration management using environment variables

### Architecture Flow

1. **Configuration Loading**: Loads Binance API credentials and ClickHouse connection settings from environment variables
2. **WebSocket Connection**: Establishes connections to `wss://stream.binance.com:443/ws` using the connector package
3. **Stream Management**: Automatically manages multiple WebSocket connections (max 1022 streams per connection) with automatic reconnection
4. **Event Processing**: Routes incoming messages to appropriate handlers (trade events, book ticker events)
5. **Data Storage**: Processes and stores trade data in ClickHouse database

### Key Features

- **Connection Pool Management**: Automatically creates new connections when stream limits are reached
- **Automatic Reconnection**: Handles connection failures with graceful reconnection and stream resubscription
- **Rate Limiting**: Respects Binance API limits (max 100 subscriptions per request)
- **Graceful Shutdown**: Handles SIGTERM/SIGINT for clean application shutdown

### Environment Variables

Required:
- `BINANCE_API_KEY`: Binance API key
- `BINANCE_SECRET_KEY`: Binance secret key

Optional:
- `BINANCE_USE_TESTNET`: Use testnet (default: true)
- `CLICKHOUSE_HOST`: ClickHouse host (default: localhost)
- `CLICKHOUSE_PORT`: ClickHouse port (default: 9000)
- `CLICKHOUSE_DATABASE`: Database name (default: default)
- `CLICKHOUSE_USERNAME`: Username (default: default)
- `CLICKHOUSE_PASSWORD`: Password (default: empty)
- `CLICKHOUSE_DEBUG`: Enable debug logging (default: false)

### Data Processing

The system processes two main event types:
- **Trade Events**: Individual trade executions with price, quantity, and order IDs
- **Book Ticker Events**: Best bid/ask price updates (currently placeholder implementation)

All events are logged using Go's structured logging (slog) for monitoring and debugging.