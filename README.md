# Alarket

A high-performance cryptocurrency market data collector built with Go and ClickHouse. Streams real-time trade data and order book updates from Binance WebSocket API with automatic connection management, reconnection handling, and optimized batch processing.

## Features

- **Real-time Data Streaming**: Subscribe to live trade events and book ticker updates from Binance
- **Automatic Connection Management**: Handles WebSocket connection pooling with automatic scaling when stream limits are reached
- **Robust Reconnection**: Graceful handling of connection failures with automatic stream resubscription
- **Optimized Batch Processing**: Collects data in batches and flushes to ClickHouse for optimal database performance
- **Clean Architecture**: Well-structured codebase following clean architecture principles for maintainability and testability
- **Historical Data Import**: Tools for importing historical trade data and file-based imports
- **Health Monitoring**: WebSocket connections use ping/pong mechanism (30-second intervals) for connection health
- **Graceful Shutdown**: Handles SIGTERM/SIGINT for clean application shutdown with final batch flush

## Requirements

- **Go**: 1.23.0 or higher
- **Docker & Docker Compose**: For running ClickHouse database
- **Binance API Keys**: Optional, only required for private endpoints (public market data doesn't require authentication)

## Quick Start

```bash
# Clone the repository
git clone https://github.com/JorryGo/alarket.git
cd alarket

# Copy environment configuration
cp .env.example .env
# Edit .env and add your configuration (optional for public data)

# Start database and application
make start
```

The application will start collecting real-time trade data from Binance and storing it in ClickHouse.

## Available Tools

Alarket provides three main tools for collecting and importing cryptocurrency market data:

### 1. Trade Collector (Real-time Data)

The primary tool for collecting live market data from Binance WebSocket API.

**Command:**
```bash
./build/trade-collector
```

**Configuration:**
This tool is fully configured via environment variables (see [Configuration](#configuration) section). No command-line arguments required.

**What it does:**
- Connects to Binance WebSocket API
- Subscribes to trade events and/or book ticker updates
- Automatically manages multiple connections when needed
- Stores data in ClickHouse with batch processing
- Handles reconnections and graceful shutdown

**Example:**
```bash
# Configure via .env file first
cp .env.example .env
# Edit .env: set SYMBOLS=BTCUSDT,ETHUSDT to collect specific pairs

# Build and run
make build
./build/trade-collector
```

### 2. Historical Trades Importer

Import historical trade data from Binance to backfill your database.

**Command:**
```bash
./build/historical-trades --symbol <SYMBOL> [flags]
```

**Required Flags:**
- `--symbol`, `-s`: Trading pair symbol (e.g., BTCUSDT, ETHUSDT)

**Optional Flags:**
- `--days`, `-d`: Number of days of historical data to fetch (default: 7)
- `--forward`, `-f`: Fetch trades forward from newest ID to fill gaps (default: false)

**What it does:**
- Checks existing data in ClickHouse
- Fetches missing historical trades from Binance API
- Default mode: fetches backward (older trades)
- Forward mode: fills gaps between newest stored trade and current time
- Respects API rate limits

**Examples:**
```bash
# Build the tool
make build-historical

# Import last 7 days of BTC trades (default)
./build/historical-trades --symbol BTCUSDT

# Import last 30 days of ETH trades
./build/historical-trades --symbol ETHUSDT --days 30

# Fill gaps in existing data (forward mode)
./build/historical-trades --symbol BTCUSDT --forward

# Short flags
./build/historical-trades -s SOLUSDT -d 14
```

### 3. File Import Tool

Import trade data from CSV files into ClickHouse.

**Command:**
```bash
./build/file-import --file <PATH> --symbol <SYMBOL>
```

**Required Flags:**
- `--file`, `-f`: Path to CSV file to import
- `--symbol`, `-s`: Trading pair symbol for the data (e.g., ETHUSDT)

**CSV Format Required:**
```csv
ID,Price,Quantity,QuoteQuantity,Timestamp,IsBuyerMaker,IsBestMatch
123456,50000.50,0.01,500.0050,1699999999000000,true,true
```

**Fields:**
- `ID`: Trade ID (string)
- `Price`: Trade price (decimal)
- `Quantity`: Trade quantity (decimal)
- `QuoteQuantity`: Quote quantity (decimal, not stored but required in CSV)
- `Timestamp`: Unix timestamp in microseconds
- `IsBuyerMaker`: Boolean (true/false)
- `IsBestMatch`: Boolean (true/false, not stored but required in CSV)

**What it does:**
- Reads CSV files with streaming processing (handles large files)
- Parses and validates trade data
- Saves in batches (100,000 trades per batch)
- Shows progress with percentage completion

**Examples:**
```bash
# Build the tool
make build-file-import

# Import a CSV file
./build/file-import --file ./data/btc_trades.csv --symbol BTCUSDT

# Import with short flags
./build/file-import -f ~/Downloads/eth_historical.csv -s ETHUSDT
```

### Build All Tools

To build all three tools at once:

```bash
make build-all
```

This creates binaries in `./build/`:
- `./build/trade-collector`
- `./build/historical-trades`
- `./build/file-import`

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/JorryGo/alarket.git
cd alarket
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Configure Environment

Create a `.env` file from the example:

```bash
cp .env.example .env
```

Edit `.env` and configure your settings (see Configuration section below).

### 4. Start ClickHouse Database

```bash
make db-up
```

This will:
- Create necessary directories
- Start ClickHouse in Docker
- Wait for database to be ready
- Initialize database schema

### 5. Build and Run

```bash
# Build the trade collector
make build

# Run the application
make run
```

Or start everything at once:

```bash
make start
```

## Configuration

All configuration is done through environment variables. You can set them in the `.env` file or export them in your shell.

### Binance Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `BINANCE_API_KEY` | Binance API key | `""` | No* |
| `BINANCE_SECRET_KEY` | Binance secret key | `""` | No* |
| `BINANCE_USE_TESTNET` | Use Binance testnet instead of production | `false` | No |

\* *API keys are only required for authenticated endpoints. Public market data streaming works without authentication.*

**Getting API Keys:**
1. Go to [Binance API Management](https://www.binance.com/en/my/settings/api-management)
2. Create a new API key
3. Enable "Enable Reading" permission (no trading permissions needed for data collection)
4. Copy the API key and Secret key to your `.env` file

### ClickHouse Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CLICKHOUSE_HOST` | ClickHouse server host | `localhost` | No |
| `CLICKHOUSE_PORT` | ClickHouse native protocol port | `9000` | No |
| `CLICKHOUSE_DATABASE` | Database name | `alarket` | No |
| `CLICKHOUSE_USERNAME` | Database username | `default` | No |
| `CLICKHOUSE_PASSWORD` | Database password | `""` | No |
| `CLICKHOUSE_DEBUG` | Enable debug logging for database queries | `false` | No |

### Application Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LOG_LEVEL` | Application log level: `debug`, `info`, `warn`, `error` | `info` | No |
| `SUBSCRIBE_TRADES` | Enable trade event subscription | `true` | No |
| `SUBSCRIBE_BOOK_TICKERS` | Enable book ticker (best bid/ask) subscription | `false` | No |
| `SYMBOLS` | Comma-separated list of symbols to collect (e.g., `BTCUSDT,ETHUSDT`). If empty, collects **all active trading pairs** | `""` (all active pairs) | No |
| `BATCH_SIZE` | Number of records to batch before flushing to ClickHouse | `10000` | No |
| `BATCH_FLUSH_TIMEOUT_MS` | Maximum time in milliseconds to wait before flushing batch | `1000` | No |

**Symbol Filtering Examples:**

```env
# Collect only specific symbols
SYMBOLS=BTCUSDT,ETHUSDT,BNBUSDT,SOLUSDT

# Collect ALL active trading pairs (default - can be 1000+ pairs!)
SYMBOLS=

# Or omit the variable entirely to collect all pairs
```

**Warning:** When `SYMBOLS` is empty, the collector will subscribe to **ALL** active trading pairs from Binance (potentially 1000+ pairs). This generates significant data volume and WebSocket connections. For production use, it's recommended to specify only the symbols you need.

## Make Commands

Alarket provides convenient Make targets for common operations:

### Build Commands
```bash
make build              # Build the trade collector
make build-historical   # Build the historical trades collector
make build-file-import  # Build the file import tool
make build-all          # Build all binaries
```

### Run Commands
```bash
make run                # Run the trade collector
make start              # Start database and application together
```

### Database Commands
```bash
make db-up              # Start ClickHouse database
make db-down            # Stop ClickHouse database
make db-reset           # Reset database (removes all data)
make db-test            # Test database connection and show status
make logs               # Show database logs
```

### Maintenance Commands
```bash
make clean              # Clean build artifacts
make help               # Show all available commands
```

## Architecture

Alarket follows Clean Architecture principles with clear separation of concerns:

### Layer Structure

```
├── cmd/                    # Application entry points
│   ├── trade-collector/   # Real-time data collector
│   ├── historical-trades/ # Historical data importer
│   └── file-import/       # File import tool
│
├── internal/
│   ├── domain/            # Domain layer (entities, interfaces)
│   │   ├── entities/      # Core business objects
│   │   ├── repositories/  # Repository abstractions
│   │   └── events/        # Domain events
│   │
│   ├── application/       # Application layer (use cases)
│   │   ├── usecases/      # Business logic
│   │   └── services/      # Application services
│   │
│   └── infrastructure/    # Infrastructure layer
│       ├── websocket/     # Generic WebSocket management
│       ├── binance/       # Binance-specific implementations
│       ├── clickhouse/    # Database implementations
│       ├── config/        # Configuration management
│       └── container/     # Dependency injection
```

### Data Flow

1. **Symbol Loading**: Fetches active trading symbols from Binance API
2. **WebSocket Connection**: Establishes managed connections with automatic scaling
3. **Event Processing**: Messages flow through clean architecture layers:
   - WebSocket → Binance Client → Event Handler → Use Cases → Repositories
4. **Batch Processing**: Data is collected in batches and flushed to ClickHouse every 1 second or when batch is full
5. **Data Storage**: Trade and book ticker data persisted to ClickHouse for analytics

### Key Technical Details

- **Connection Pool Management**: Automatically creates new connections when reaching the 1022 streams per connection limit
- **Rate Limiting**: Respects Binance API limits (max 100 subscriptions per request)
- **Asynchronous Writes**: Uses goroutines for non-blocking database writes
- **Connection Health**: 30-second ping/pong intervals for connection monitoring
- **Graceful Shutdown**: 10-second timeout for final batch flush on termination

## Database Schema

### Trades Table

Stores individual trade executions:

```sql
CREATE TABLE trades (
    event_time DateTime64(3),
    symbol String,
    trade_id UInt64,
    price Decimal(18, 8),
    quantity Decimal(18, 8),
    buyer_order_id UInt64,
    seller_order_id UInt64,
    trade_time DateTime64(3),
    is_buyer_maker Bool,
    date Date
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (symbol, trade_time);
```

### Book Tickers Table

Stores best bid/ask price updates:

```sql
CREATE TABLE book_tickers (
    update_id UInt64,
    symbol String,
    best_bid_price Decimal(18, 8),
    best_bid_qty Decimal(18, 8),
    best_ask_price Decimal(18, 8),
    best_ask_qty Decimal(18, 8),
    event_time DateTime64(3),
    date Date
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (symbol, event_time);
```

## Security Best Practices

### API Key Management

1. **Never commit `.env` file**: The `.env` file is in `.gitignore` to prevent accidental commits
2. **Read-only permissions**: Your Binance API keys only need "Enable Reading" permission for data collection
3. **IP restrictions**: Consider restricting API keys to your server's IP address in Binance settings
4. **Rotate keys regularly**: Periodically rotate your API keys for enhanced security
5. **Use testnet for development**: Set `BINANCE_USE_TESTNET=true` when developing or testing

### Database Security

1. **Change default password**: Set a strong `CLICKHOUSE_PASSWORD` for production deployments
2. **Network isolation**: Run ClickHouse in a private network, not exposed to the internet
3. **Backup regularly**: Implement regular database backups for production data

### Production Deployment

1. **Use environment variables**: Never hardcode secrets in source code
2. **Secure connection strings**: Use TLS/SSL for database connections in production
3. **Monitor logs**: Set `LOG_LEVEL=warn` or `error` in production to reduce log verbosity
4. **Rate limiting**: Monitor API usage to stay within Binance rate limits

## Performance Tuning

### Batch Processing

Adjust batch settings based on your data volume:

```env
# High-volume settings (more throughput, higher memory)
BATCH_SIZE=50000
BATCH_FLUSH_TIMEOUT_MS=5000

# Low-latency settings (lower latency, more writes)
BATCH_SIZE=1000
BATCH_FLUSH_TIMEOUT_MS=100
```

### Database Optimization

ClickHouse is optimized for analytical queries. For better performance:

- Partition tables by month (`PARTITION BY toYYYYMM(date)`)
- Use appropriate data types (Decimal for prices, UInt64 for IDs)
- Create materialized views for common aggregations
- Compress older partitions

## Troubleshooting

### Connection Issues

**Problem**: WebSocket connection keeps disconnecting

**Solutions**:
- Check your internet connection stability
- Verify Binance API status at [Binance Status](https://www.binance.com/en/support/announcement)
- Ensure you're not hitting rate limits
- Check logs with `LOG_LEVEL=debug`

### Database Issues

**Problem**: Can't connect to ClickHouse

**Solutions**:
```bash
# Check if ClickHouse is running
docker-compose ps

# Check ClickHouse logs
make logs

# Reset database
make db-reset
```

**Problem**: High memory usage

**Solutions**:
- Reduce `BATCH_SIZE`
- Reduce `BATCH_FLUSH_TIMEOUT_MS`
- Monitor with `CLICKHOUSE_DEBUG=true`

### Data Issues

**Problem**: No data being collected

**Solutions**:
- Check if subscriptions are enabled: `SUBSCRIBE_TRADES=true`
- Verify you're subscribed to active symbols
- Check application logs for errors
- Test database connection: `make db-test`

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Development Setup

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Code Style

- Follow Go best practices and idioms
- Use `gofmt` for code formatting
- Add tests for new functionality
- Update documentation as needed

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Binance API](https://binance-docs.github.io/apidocs/) for market data
- [ClickHouse](https://clickhouse.com/) for high-performance analytics database
- [gorilla/websocket](https://github.com/gorilla/websocket) for WebSocket implementation

## Support

If you encounter any issues or have questions:

1. Check the [Troubleshooting](#troubleshooting) section
2. Search [existing issues](https://github.com/JorryGo/alarket/issues)
3. Create a [new issue](https://github.com/JorryGo/alarket/issues/new) with detailed information

## Roadmap

- [ ] Support for additional exchanges (Coinbase, Kraken, etc.)
- [ ] Real-time analytics dashboard
- [ ] Advanced data aggregation and indicators
- [ ] Alerting system for price movements
- [ ] REST API for data access
- [ ] Time-series data compression
- [ ] Multi-instance deployment support

---

**Disclaimer**: This software is for educational and research purposes. Cryptocurrency trading involves risk. Always do your own research and use this software responsibly.
