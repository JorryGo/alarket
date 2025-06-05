package clickhouse

import (
	"alarket/internal/binance/events"
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/rs/zerolog/log"
	"net"
	"time"
)

type Service struct {
	conn driver.Conn
}

func NewService() (*Service, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", `localhost`, 123)},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "default",
		},
		DialContext: func(ctx context.Context, addr string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "tcp", addr)
		},
		Debug: true,
		Debugf: func(format string, v ...interface{}) {
			fmt.Printf(format, v)
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:      time.Duration(10) * time.Second,
		MaxOpenConns:     5,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Duration(10) * time.Minute,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
		BlockBufferSize:  10,
	})
	if err != nil {
		return nil, err
	}

	return &Service{
		conn: conn,
	}, nil
}

func (c *Service) WriteTradeEvent(event events.TradeEvent) {
	query := `INSERT INTO trade (symbol, event_time, price, quantity, buyer_order_id, seller_order_id, trade_id, buyer_market_maker, ignore) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	err := c.conn.Exec(
		context.Background(),
		query,
		event.Symbol,
		event.Time,
		event.Price,
		event.Quantity,
		event.BuyerOrderID,
		event.SellerOrderID,
		event.TradeID,
		event.IsBuyerMaker,
		event.Placeholder,
	)

	if err != nil {
		log.Err(err)
	}

}
