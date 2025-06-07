-- Test database connection and show tables
SHOW DATABASES;

USE alarket;

SHOW TABLES;

-- Show table schemas
DESCRIBE trades;

DESCRIBE book_tickers;

-- Count records
SELECT 'Total trades:', count() FROM trades;
SELECT 'Total book_tickers:', count() FROM book_tickers;

-- Show latest trades (if any)
SELECT 'Latest 5 trades:' as info;
SELECT symbol, price, quantity, trade_time 
FROM trades 
ORDER BY trade_time DESC 
LIMIT 5;