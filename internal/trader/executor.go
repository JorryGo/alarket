package trader

import (
	"context"
	"fmt"
	"github.com/adshao/go-binance/v2"
)

const apiKey = "ntCyqh0qiuEGtg9jH4EOtIoeSv8ETOjiHdeHjs0zNgwpckL5flXpigOIT5teYmzv"
const secretKey = "0x26WvMtZietFtv5nGvqQnM2WrIlXTSzQGiCWej045lus55OKo5bemF1HHPQ3Vtn"

type Executor struct {
	client *binance.Client
}

func InitExecutor() *Executor {
	return &Executor{
		client: binance.NewClient(apiKey, secretKey),
	}
}

func (e *Executor) BuyMarket(symbol string, quantity float64) (float64, error) {
	order, err := e.client.NewCreateOrderService().Symbol("BTCUSDT").
		Side(binance.SideTypeBuy).Type(binance.OrderTypeMarket).
		Quantity("1").Do(context.Background())

	res, err2 := e.client.NewGetAccountService().Do(context.Background())

	fmt.Println(order, err2, res)

	if err != nil {
		return 0, err
	}

	return 1.4, nil
}
