# Clean Architecture for Alarket Trade Collector

## Overview

This project has been refactored to follow Clean Architecture principles, ensuring a clear separation of concerns, testability, and maintainability.

## Architecture Layers

### 1. Domain Layer (`internal/domain/`)
The core business logic layer, independent of any external frameworks or infrastructure.

- **Entities** (`entities/`): Core business objects (Trade, BookTicker, Symbol)
- **Repository Interfaces** (`repositories/`): Abstractions for data persistence
- **Services Interfaces** (`services/`): Abstractions for external services
- **Domain Events** (`events/`): Business events

### 2. Application Layer (`internal/application/`)
Contains application-specific business rules and orchestrates the flow of data.

- **Use Cases** (`usecases/`): Application-specific business logic
- **DTOs** (`dto/`): Data Transfer Objects for external communication
- **Services** (`services/`): Application services like EventHandler

### 3. Infrastructure Layer (`internal/infrastructure/`)
External frameworks, tools, and implementations.

- **WebSocket** (`websocket/`): Generic WebSocket connection management
- **Binance** (`binance/`): Binance-specific implementations
- **ClickHouse** (`clickhouse/`): Database repository implementations
- **Config** (`config/`): Configuration management
- **Container** (`container/`): Dependency injection container

### 4. Interfaces Layer (`internal/interfaces/`)
Entry points to the application (currently unused but reserved for future REST/gRPC APIs).

## Key Design Principles

### 1. Dependency Inversion
- Domain layer defines interfaces
- Infrastructure layer implements these interfaces
- Dependencies flow inward (Infrastructure → Application → Domain)

### 2. Single Responsibility
- Each module has one reason to change
- Clear separation between business logic and infrastructure concerns

### 3. Interface Segregation
- Small, focused interfaces
- Clients depend only on the methods they use

### 4. Dependency Injection
- All dependencies injected through constructors
- Centralized in the container package

## Data Flow

1. **WebSocket Message Received** → WebSocket Manager
2. **Message Handling** → Binance Client → Event Handler
3. **Business Logic** → Use Cases (ProcessTradeEvent, ProcessBookTicker)
4. **Data Persistence** → Repository implementations → ClickHouse

## Benefits

1. **Testability**: Each layer can be tested independently with mocks
2. **Maintainability**: Clear structure and separation of concerns
3. **Flexibility**: Easy to swap implementations (e.g., different exchange or database)
4. **Business Logic Protection**: Core logic isolated from external changes

## Configuration

All configuration is centralized in environment variables:
- `BINANCE_API_KEY`: Binance API key (required)
- `BINANCE_SECRET_KEY`: Binance secret key (required)
- `BINANCE_USE_TESTNET`: Use testnet (default: true)
- `CLICKHOUSE_*`: Database configuration
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `SUBSCRIBE_TRADES`: Enable trade subscription (default: true)
- `SUBSCRIBE_BOOK_TICKERS`: Enable book ticker subscription (default: true)

## Future Enhancements

1. Add REST/gRPC APIs in the interfaces layer
2. Implement caching layer
3. Add metrics and monitoring
4. Implement event sourcing for audit trail
5. Add more sophisticated error handling and recovery strategies