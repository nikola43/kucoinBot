package main

import (
	"fmt"
	"github.com/Kucoin/kucoin-go-sdk"
	"github.com/fatih/color"
	"log"
	"math"
	"math/rand"
	"strconv"

	"strings"
)

func main() {
	initialPrice := 0.0
	initialBuyPrice := 0.0
	highPrice := 0.0
	stopLossPrice := 0.0
	selectedSymbolBalance := 0.0
	selectedPairBalance := 0.0
	sellQuantity := 0.0
	buyQuantity := 0.0
	orderId := ""
	coinExist := false
	coinName := "go"
	pairCoinName := "usdt"
	selectedCoin := strings.ToUpper(coinName)
	selectedPair := strings.ToUpper(pairCoinName)
	selectedSymbol := selectedCoin + "-" + selectedPair
	fmt.Println(selectedSymbol)

	kucoin.DebugMode = true

	// API key version 2.0
	kucoinService := kucoin.NewApiService(
		kucoin.ApiKeyOption("605cd5d0186f170006cef54a"),
		kucoin.ApiSecretOption("cc2b6051-f7c2-4bef-8d1c-171ff0b88b46"),
		kucoin.ApiPassPhraseOption("-ArdBoard21"),
		kucoin.ApiKeyVersionOption("ApiKeyVersionV2"),
	)

	// INIT WEBSOCKET ---------------------------------------
	rsp, err := kucoinService.WebSocketPublicToken()
	if err != nil {
		fmt.Println(err.Error())
	}

	tk := &kucoin.WebSocketTokenModel{}
	if err := rsp.ReadData(tk); err != nil {
		// Handle error
		return
	}

	c := kucoinService.NewWebSocketClient(tk)
	mc, ec, err := c.Connect()
	if err != nil {
		// Handle error
		return
	}
	//-------------------------------------------------------

        fmt.Print("Enter text: ")
        var input string
        fmt.Scanln(&input)
        selectedCoin = strings.ToUpper(input)               
        selectedPair = strings.ToUpper(pairCoinName)
        selectedSymbol = selectedCoin + "-" + selectedPair


	selectedSymbolBalance = parsePriceToFloat(getBalanceByCoin(kucoinService, coinName))
	selectedPairBalance = parsePriceToFloat(getBalanceByCoin(kucoinService, pairCoinName))
	sellQuantity = math.Round(selectedSymbolBalance - (selectedSymbolBalance * 2 / 100))

	fmt.Println("selectedSymbolBalance")
	fmt.Println(selectedSymbolBalance)
	fmt.Println("selectedPairBalance")
	fmt.Println(selectedPairBalance)
	fmt.Println("sellQuantity")
	fmt.Println(sellQuantity)

	// Check if selected symbol exists
	for ok := true; ok; ok = coinExist == false {
		ticker := getSymbolTicker(kucoinService, selectedSymbol)
		if ticker != nil {
			coinExist = true
			initialPrice = parsePriceToFloat(ticker.BestAsk)
			fmt.Println(ticker.BestAsk)
			fmt.Println(initialPrice)

			// todo cambiar por precio compra initialBuyPrice
			// todo comprar
			buyQuantity = math.Round((selectedPairBalance / initialPrice) - ((selectedPairBalance / initialPrice) * 2 / 100))
			orderResult := createMarketOrder(kucoinService, "buy", selectedSymbol, parsePriceToString(buyQuantity))
			orderId = orderResult.OrderId
			fmt.Println(orderId)

			order := getOrder(kucoinService, orderId)
			if order.IsActive == false {
				tickerAfterBuy := getSymbolTicker(kucoinService, selectedSymbol)
				initialBuyPrice = parsePriceToFloat(tickerAfterBuy.BestAsk)
				selectedSymbolBalance = parsePriceToFloat(getBalanceByCoin(kucoinService, coinName))
				sellQuantity = math.Round(selectedSymbolBalance - (selectedSymbolBalance * 2 / 100))

			}

		} else {
			fmt.Println("NO EXISTE")
		}

	}

	ch1 := kucoin.NewSubscribeMessage("/market/ticker:"+selectedCoin+"-"+selectedPair, false)
	if err := c.Subscribe(ch1); err != nil {
		fmt.Println(err.Error())
	}
	for {
		select {
		case err := <-ec:
			fmt.Println("EERR")
			c.Stop() // Stop subscribing the WebSocket feed
			fmt.Printf("Error: %s", err.Error())
			// Handle error
			return
		case msg := <-mc:
			// log.Printf("Received: %s", kucoin.ToJsonString(m))
			ticker := &kucoin.TickerLevel1Model{}
			if err := msg.ReadData(ticker); err != nil {
				fmt.Printf("Failure to read: %s", err.Error())
				return
			}
			// ---------------------------------------------------------------------------------------------------------
			fmt.Printf("Ticker: %s, %s\n", ticker.BestBid, ticker.BestAsk)

			// new target
			currentPrice := parsePriceToFloat(ticker.BestBid)
			if currentPrice > initialBuyPrice && currentPrice > highPrice {
				highPrice = parsePriceToFloat(ticker.BestBid)
				color.Yellow("Nuevo precio más alto")
				stopPrice := highPrice - (highPrice * 5 / 100)
				sellPrice := highPrice - (highPrice * 10 / 100)

				fmt.Println("sellPrice")
				fmt.Println(sellPrice)
				fmt.Println("stopPrice")
				fmt.Println(stopPrice)

				cancelOrder(kucoinService, orderId)
				orderResult := createTakeProfitOrder(
					kucoinService,
					selectedSymbol,
					parsePriceToString(sellQuantity),
					parsePriceToString(stopPrice),
					parsePriceToString(sellPrice))

				orderId = orderResult.OrderId
			}

			// STOP LOSS SELL
			if currentPrice <= stopLossPrice {
				color.Red("STOP LOSS")
				cancelOrder(kucoinService, orderId)
				orderResult := createMarketOrder(kucoinService, "sell", selectedSymbol, parsePriceToString(sellQuantity))
				orderId = orderResult.OrderId
				return
			}

			// ---------------------------------------------------------------------------------------------------------
			if err = c.Subscribe(ch1); err != nil {
				fmt.Printf("Error: %s", err.Error())
				// Handle error
				return
			}
		}
	}

}

func getBalanceByCoin(kucoinService *kucoin.ApiService, currency string) string {
	balance := ""
	accounts := kucoin.AccountsModel{}
	b, err := kucoinService.Accounts(currency, "trade")
	if err != nil {
		fmt.Println(err.Error())
	}

	err = b.ReadData(&accounts)
	if err != nil {
		fmt.Println(err.Error())
	}

	if len(accounts) > 0 {
		balance = accounts[0].Available
		log.Printf("Available balance: %s %s => %s", accounts[0].Type, accounts[0].Currency, accounts[0].Available)
	}

	return balance
}

func parsePriceToFloat(price string) float64 {
	f, _ := strconv.ParseFloat(price, 8)
	return f
}
func parsePriceToString(price float64) string {
	s := fmt.Sprintf("%.5f", price)
	return s
}

func createMarketOrder(kucoinService *kucoin.ApiService, side, symbol, size string) *kucoin.CreateOrderResultModel {
	oid := strconv.FormatInt(int64(rand.Intn(99999999)), 10)

	order := &kucoin.CreateOrderModel{
		ClientOid: oid,
		Side:      side,
		Symbol:    symbol,
		Type:      "market",
		Size:      size,
	}

	createOrderResult, err := kucoinService.CreateOrder(order)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	marketOrder := &kucoin.CreateOrderResultModel{}
	err = createOrderResult.ReadData(marketOrder)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	fmt.Println(marketOrder)
	return marketOrder
}

func createTakeProfitOrder(kucoinService *kucoin.ApiService, symbol, size, stopPrice, price string) *kucoin.CreateOrderResultModel {
	createOrderResultModel := &kucoin.CreateOrderResultModel{}
	oid := strconv.FormatInt(int64(rand.Intn(99999999)), 10)

	order := &kucoin.CreateOrderModel{
		ClientOid: oid,
		Side:      "sell",
		Symbol:    symbol,
		Stop:      "loss",
		StopPrice: stopPrice,
		Price:     price,
		Size:      size,
	}

	createOrderResult, err := kucoinService.CreateOrder(order)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	err = createOrderResult.ReadData(createOrderResultModel)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	fmt.Println(createOrderResultModel)
	return createOrderResultModel
}

func createBuyOrder(kucoinService *kucoin.ApiService, symbol string, size string, price string) *kucoin.CreateOrderResultModel {
	createOrderResultModel := &kucoin.CreateOrderResultModel{}
	oid := strconv.FormatInt(int64(rand.Intn(99999999)), 10)

	order := &kucoin.CreateOrderModel{
		ClientOid: oid,
		Side:      "buy",
		Symbol:    symbol,
		Price:     price,
		Size:      size,
	}

	createOrderResult, err := kucoinService.CreateOrder(order)
	if err != nil {
		// fmt.Println(err.Error())
		return nil
	}

	err = createOrderResult.ReadData(createOrderResultModel)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	fmt.Println(createOrderResultModel)
	return createOrderResultModel
}

func getOrder(kucoinService *kucoin.ApiService, orderId string) *kucoin.OrderModel {
	order := &kucoin.OrderModel{}
	getOrderResult, err := kucoinService.Order(orderId)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	err = getOrderResult.ReadData(order)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	return order
}

func cancelOrder(kucoinService *kucoin.ApiService, orderId string) bool {
	cancelOrderModel := &kucoin.CancelOrderResultModel{}

	order := getOrder(kucoinService, orderId)
	if order.IsActive {
		cancelOrderResult, err := kucoinService.CancelOrder(orderId)
		if err != nil {
			fmt.Println(err.Error())
			return false
		}
		err = cancelOrderResult.ReadData(cancelOrderModel)
		if err != nil {
			fmt.Println(err.Error())
			return false
		}
	}


	return true
}

func getSymbolTicker(kucoinService *kucoin.ApiService, selectedSymbol string) *kucoin.TickerLevel1Model {
	apiResponse, err := kucoinService.TickerLevel1(selectedSymbol)
	if err != nil {
		// fmt.Println(err)
	}

	ticker := &kucoin.TickerLevel1Model{}
	err = apiResponse.ReadData(ticker)
	if err != nil {
		// fmt.Println(err.Error())
		return nil
	}
	return ticker
}
