package main

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/adshao/go-binance/v2"
)

const (
	symbol = "ETHUSDT"
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

	// Get account information
	account, err := client.NewGetAccountService().Do(context.Background())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Initialize order parameters
	var buyOrder *binance.CreateOrderResponse
	var sellOrder *binance.CreateOrderResponse
	var stopLossOrder *binance.CreateOrderResponse

	// Wait for first candle to close
	// time.Sleep(60 * time.Second)

	// Start loop to check for price breakouts
	for {

		var currentPrice float64
		var buyPrice float64

		// Get current klines
		klines, err := client.NewKlinesService().Symbol(symbol).Interval("1m").Limit(1).Do(context.Background())
		if err != nil {
			fmt.Println(err)
			continue
		}

		for _, kline := range klines {
			float := kline.Close
			currentPrice, err = strconv.ParseFloat(float, 64)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Printf("Current Price: %s\n", float)
		}

		var quantity string
		var buyQty float64
		var rounded float64
		var buyFirst bool

		// Find available balance for the asset being traded
		for _, balance := range account.Balances {
			if balance.Asset == "USDT" {

				fmt.Printf("USDT Balance: %v\n", balance.Free)

				usdtFloat, err := strconv.ParseFloat(balance.Free, 64)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
				}

				if usdtFloat > 1 {
					buyFirst = true
					feePercentage := 1.00
					fee := usdtFloat * (feePercentage / 100)
					buyQty = (usdtFloat - fee) / currentPrice
					rounded = math.Round(buyQty*10000) / 10000
					quantity = balance.Free
					break
				} else {
					continue
				}
			}

			if balance.Asset == "ETH" {
				fmt.Printf("ETH Balance: %v\n", balance.Free)

				ethFloat, err := strconv.ParseFloat(balance.Free, 64)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
				}

				if ethFloat > 0.01 {
					buyFirst = false
					buyPrice, err = getBuyPrice(*client)
					if err != nil {
						fmt.Println(err)
						continue
					}
					sellOrder = nil
					feePercentage := 1.00
					fee := ethFloat * (feePercentage / 100)
					rounded = math.Round((ethFloat-fee)*10000) / 10000
					quantity = balance.Free
					break
				}
			}
		}

		lastDigit := int(math.Round(currentPrice)) % 10

		fmt.Printf("Current Price: %v\n", currentPrice)
		fmt.Printf("Quantity: %v\n", quantity)
		fmt.Printf("Last Digit: %v\n", lastDigit)
		fmt.Printf("Rounded Buy Quantity: %v\n", rounded)
		fmt.Printf("Last Buy Price: %v\n", buyPrice)
		fmt.Printf("Last Buy Qty: %v\n", buyQty)

		if (lastDigit == 1 || lastDigit == 2) && buyOrder == nil && buyFirst {
			fmt.Printf("Buy order\n")

			// Place a market order to buy at current price
			order, err := client.NewCreateOrderService().
				Symbol(symbol).
				Side(binance.SideTypeBuy).
				Type(binance.OrderTypeMarket).
				Quantity(fmt.Sprintf("%.8f", rounded)).
				Do(context.Background())
			if err != nil {
				fmt.Println(err)
				time.Sleep(60 * time.Second)
				continue
			}
			buyOrder = order
			sellOrder = nil
			buyPrice = currentPrice

			fmt.Printf("Buy order placed: %v\n", order)
		}

		if (lastDigit == 8 || lastDigit == 9) && sellOrder == nil && buyPrice < currentPrice {

			// Place a limit order to sell at current price
			order, err := client.NewCreateOrderService().
				Symbol(symbol).
				Side(binance.SideTypeSell).
				Type(binance.OrderTypeMarket).
				Quantity(fmt.Sprint(rounded)).
				Do(context.Background())
			if err != nil {
				fmt.Println(err)
				time.Sleep(60 * time.Second)
				continue
			}

			sellOrder = order
			buyOrder = nil
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
					Quantity(fmt.Sprint(rounded)).
					Do(context.Background())
				if err != nil {
					fmt.Println(err)
					continue
				}
				stopLossOrder = order
				fmt.Printf("Stop loss order placed: %v\n", order)
			}
		}

		fmt.Println("--------")
		// Wait for next minute
		time.Sleep(60 * time.Second)
	}
}

func getBuyPrice(c binance.Client) (float64, error) {
	// Fetch the order book for the symbol
	depth, err := c.NewDepthService().Symbol(symbol).Do(context.Background())
	if err != nil {
		fmt.Println("Error:", err)
		return 0.00, err
	}

	lastOrderPrice, err := strconv.ParseFloat(depth.Bids[0].Price, 64)
	if err != nil {
		fmt.Println("Error:", err)
		return 0.00, err
	}

	// Retrieve the last buy order price from the order book
	return lastOrderPrice, nil
}
