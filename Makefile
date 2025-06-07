.PHONY: build build-historical run db-up db-down db-reset db-test logs clean help

# Build the trade collector application
build:
	mkdir -p ./build && go build -o ./build/trade-collector cmd/trade-collector/main.go

# Build the historical trades collector
build-historical:
	mkdir -p ./build && go build -o ./build/historical-trades cmd/historical-trades/main.go

# Build all binaries
build-all: build build-historical

# Run the application
run: build
	./build/trade-collector

# Start ClickHouse database
db-up:
	docker-compose up -d clickhouse
	@echo "Waiting for ClickHouse to be ready..."
	@until docker-compose exec clickhouse clickhouse-client --query "SELECT 1" > /dev/null 2>&1; do \
		echo "Waiting for ClickHouse..."; \
		sleep 2; \
	done
	@echo "ClickHouse is ready!"

# Stop ClickHouse database
db-down:
	docker-compose down

# Reset database (remove all data)
db-reset:
	docker-compose down -v
	docker-compose up -d clickhouse
	@echo "Database reset complete"

# Test database connection and show status
db-test:
	docker-compose exec clickhouse clickhouse-client --database=alarket --multiquery < scripts/test-db.sql

# Show database logs
logs:
	docker-compose logs -f clickhouse

# Clean build artifacts
clean:
	rm -rf ./build

# Start everything (database + application)
start: db-up
	@echo "Starting application..."
	@sleep 3
	$(MAKE) run

# Show help
help:
	@echo "Available commands:"
	@echo "  build              - Build the trade collector application"
	@echo "  build-historical   - Build the historical trades collector"
	@echo "  build-all          - Build all binaries"
	@echo "  run                - Run the trade collector application"
	@echo "  db-up              - Start ClickHouse database"
	@echo "  db-down            - Stop ClickHouse database"
	@echo "  db-reset           - Reset database (remove all data)"
	@echo "  db-test            - Test database connection and show status"
	@echo "  logs               - Show database logs"
	@echo "  clean              - Clean build artifacts"
	@echo "  start              - Start database and application"
	@echo "  help               - Show this help"