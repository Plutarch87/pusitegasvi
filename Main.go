package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/adshao/go-binance/v2"
)

const (
	symbol   = "BTCUSDT"
	quantity = 0.001
)

func main() {
	envErr := godotenv.Load(".env")
	if envErr != nil {
		fmt.Println("Could not load .env file")
	}

	apiKey, exists := os.LookupEnv("BINANCE_API_KEY")
	if !exists {
		fmt.Println("API key doesn't exist")
		os.Exit(1)
	}
	secretKey, exists := os.LookupEnv("BINANCE_SECRET_KEY")
	if !exists {
		fmt.Println("API secret doesn't exist")
		os.Exit(1)
	}
	// Enable use of TestNet
	binance.UseTestnet = false

	// Initialize Binance client
	client := binance.NewClient(apiKey, secretKey)

	// Initialize order parameters
	var buyOrder *binance.CreateOrderResponse
	var sellOrder *binance.CreateOrderResponse
	var stopLossOrder *binance.CreateOrderResponse

	// Wait for first candle to close
	time.Sleep(60 * time.Second)

	// Start loop to check for price breakouts
	for {
		// Get current klines
		klines, err := client.NewKlinesService().Symbol(symbol).Interval("1m").Limit(20).Do(context.Background())
		if err != nil {
			fmt.Println(err)
			continue
		}
		// Calculate current price and 20-period moving average
		var currentPrice float64
		var ma20 float64

		for _, kline := range klines {
			float := kline.Close
			currentPrice, err := strconv.ParseFloat(float, 64)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Printf("Current price: %s\n", float)

			ma20 += currentPrice
		}

		ma20 = (ma20 / 20)
		fmt.Printf("Current MA20: %v\n", ma20)

		// Check for buy signal (price breaks above 20-period MA)
		if currentPrice > ma20 && buyOrder == nil {
			fmt.Println("Buy?")
			// Place a market order to buy at current price
			order, err := client.NewCreateOrderService().
				Symbol(symbol).
				Side(binance.SideTypeBuy).
				Type(binance.OrderTypeMarket).
				Quantity(fmt.Sprintf("%.8f", quantity)).
				Do(context.Background())
			if err != nil {
				fmt.Println(err)
				continue
			}
			buyOrder = order
			fmt.Printf("Buy order placed: %v\n", order)
		}

		// Check for sell signal (price breaks below 20-period MA)
		if currentPrice < ma20 && buyOrder != nil && sellOrder == nil {
			fmt.Println("if lesser")

			// Place a limit order to sell at current price
			order, err := client.NewCreateOrderService().
				Symbol(symbol).
				Side(binance.SideTypeSell).
				Type(binance.OrderTypeMarket).
				Quantity(fmt.Sprintf("%.8f", quantity)).
				Do(context.Background())
			if err != nil {
				fmt.Println(err)
				continue
			}
			sellOrder = order
			fmt.Printf("Sell order placed: %v\n", order)
		}

		// Check for stop loss (price falls below 5% of buy price)
		if buyOrder != nil && stopLossOrder == nil {
			float2 := buyOrder.Price
			buyOrderPrice, err := strconv.ParseFloat(float2, 64)
			if err != nil {
				fmt.Println(err)
				continue
			}

			stopLoss := big.NewFloat(currentPrice).Cmp(big.NewFloat(buyOrderPrice * 0.95))
			if stopLoss < 0 {
				// Place a market order to sell at current price
				order, err := client.NewCreateOrderService().
					Symbol(symbol).
					Side(binance.SideTypeSell).
					Type(binance.OrderTypeStopLoss).
					Quantity(fmt.Sprintf("%.8f", quantity)).
					Do(context.Background())
				if err != nil {
					fmt.Println(err)
					continue
				}
				stopLossOrder = order
				fmt.Printf("Stop loss order placed: %v\n", order)
			}
		}

		// Wait for next minute
		time.Sleep(60 * time.Second)
	}
}
