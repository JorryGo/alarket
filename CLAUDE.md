# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Requirements

**Always place compiled binaries in `./build`, never in the repo root.  
Every `go build` command MUST use `-o ./build/<name>` and create the folder if it doesn't exist.**

## Quick Start

Start the database and application:
```bash
make start
```

Or step by step:
```bash
# Start ClickHouse database
make db-up

# Build and run the application
make run
```

## Build Commands

Build the trade collector:
```bash
mkdir -p ./build && go build -o ./build/trade-collector cmd/trade-collector/main.go
```

Or using make:
```bash
make build
```

## Database Management

Start ClickHouse:
```bash
make db-up
```

Stop ClickHouse:
```bash
make db-down
```

Reset database (remove all data):
```bash
make db-reset
```

View database logs:
```bash
make logs
```

## Project Architecture

This is a Go-based cryptocurrency trading data collector that follows Clean Architecture principles. It streams real-time trade data from Binance WebSocket API and stores it in ClickHouse database.

### Clean Architecture Layers

#### Domain Layer (`internal/domain/`)
- **Entities**: Core business objects (Trade, BookTicker, Symbol)
- **Repositories**: Abstractions for data persistence
- **Services**: Abstractions for external services
- **Events**: Domain events

#### Application Layer (`internal/application/`)
- **Use Cases**: Application-specific business logic
- **DTOs**: Data Transfer Objects
- **Services**: Application services (EventHandler)

#### Infrastructure Layer (`internal/infrastructure/`)
- **WebSocket**: Generic WebSocket connection management
- **Binance**: Exchange-specific implementations
- **ClickHouse**: Database repository implementations
- **Config**: Centralized configuration
- **Container**: Dependency injection

#### Interfaces Layer (`internal/interfaces/`)
- Reserved for future REST/gRPC APIs

### Architecture Flow

1. **Initialization**: Container creates all dependencies with proper injection
2. **Symbol Loading**: Fetches active trading symbols from Binance
3. **WebSocket Connection**: Establishes managed connections with automatic scaling
4. **Event Processing**: Messages flow through clean architecture layers:
   - WebSocket → Binance Client → Event Handler → Use Cases → Repositories
5. **Data Storage**: Trade and book ticker data persisted to ClickHouse

### Key Features

- **Connection Pool Management**: Automatically creates new connections when stream limits are reached
- **Automatic Reconnection**: Handles connection failures with graceful reconnection and stream resubscription
- **Rate Limiting**: Respects Binance API limits (max 100 subscriptions per request)
- **Batch Processing**: Collects data in batches and flushes to ClickHouse every 200ms or when batch is full for optimal database performance
- **Graceful Shutdown**: Handles SIGTERM/SIGINT for clean application shutdown with final batch flush

### Environment Variables

Optional:
- `BINANCE_API_KEY`: Binance API key (required only for private endpoints)
- `BINANCE_SECRET_KEY`: Binance secret key (required only for private endpoints)
- `BINANCE_USE_TESTNET`: Use testnet (default: false)
- `CLICKHOUSE_HOST`: ClickHouse host (default: localhost)
- `CLICKHOUSE_PORT`: ClickHouse port (default: 9000)
- `CLICKHOUSE_DATABASE`: Database name (default: alarket)
- `CLICKHOUSE_USERNAME`: Username (default: default)
- `CLICKHOUSE_PASSWORD`: Password (default: empty)
- `CLICKHOUSE_DEBUG`: Enable debug logging (default: false)
- `LOG_LEVEL`: Application log level - debug, info, warn, error (default: info)
- `SUBSCRIBE_TRADES`: Enable trade event subscription (default: true)
- `SUBSCRIBE_BOOK_TICKERS`: Enable book ticker subscription (default: true)
- `BATCH_SIZE`: Number of records to batch before flushing to ClickHouse (default: 1000)
- `BATCH_FLUSH_TIMEOUT_MS`: Maximum time in milliseconds to wait before flushing batch (default: 200)

### Data Processing

The system processes two main event types:
- **Trade Events**: Individual trade executions with price, quantity, and order IDs
- **Book Ticker Events**: Best bid/ask price updates

All events are:
1. Received via WebSocket connections
2. Parsed and validated in the application layer
3. Processed through use cases with business logic
4. Persisted to ClickHouse via repository implementations

The architecture ensures:
- **Testability**: Each layer can be tested independently
- **Maintainability**: Clear separation of concerns
- **Scalability**: Automatic connection management for high throughput
- **Reliability**: Graceful error handling and recovery