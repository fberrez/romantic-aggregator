package bitfinex

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/fberrez/romantic-aggregator/currency"
	"github.com/fberrez/romantic-aggregator/websocket"
)

const (
	Book   string = "book"
	Trade  string = "trades"
	Ticker string = "ticker"
)

var (
	uri url.URL = url.URL{Scheme: "wss", Host: "api.bitfinex.com", Path: "/ws"}
)

// Initializes the Bitfinex struct
func (b *Bitfinex) Initialize(kafkaChannel chan interface{}) error {
	log.Println("[Bitfinex] Initializing Bitfinex")
	b.KafkaChannel = kafkaChannel
	b.Proxy = &websocket.Proxy{
		Label: "Bitfinex",
	}

	return b.Proxy.Initialize(uri, kafkaChannel)
}

// Starts the goroutine Listen and the loop
// which will send messages present in the queue
// and will wait for the SIGINT
func (b *Bitfinex) Start(testing bool) error {
	err := b.IsClean()

	// If the Bitfinex is not clean (see (*Bitfinex)IsClean definition),
	// the process returns an error
	if err != nil {
		return errors.New(fmt.Sprintf("[Bitfinex] Bitfinex structure needs to be properly initialized before to continue.\n\tErr: %v", err))
	}

	go b.ListenResponse()
	b.Proxy.Start(testing)

	return nil
}

// Listens the Bitfinex's websocket and processes datas he receives
// before to send it to a kafka producer
func (b *Bitfinex) ListenResponse() {
	for {
		select {
		case response := <-b.Proxy.ResponseChannel:
			parsedResponse, err := MakeResponse(response)

			if err != nil {
				log.Printf("[Bitfinex] Error occured while listening Bitfinex Websocket: %v", err)
				break
			}

			if parsedResponse != nil {
				log.Printf("[Bitfinex] Sending Message %v to Kafka from Bitfinex", parsedResponse)
				b.KafkaChannel <- parsedResponse
			}
		}
	}
}

// Builds a new message to send and adds it to the queue
func (b *Bitfinex) SendMessage(aEvent string, symbols []string, channels []string) error {
	for _, symbol := range symbols {
		for _, channel := range channels {
			message := Message{
				Event:   aEvent,
				Symbol:  symbol,
				Channel: channel,
			}

			marshalledMessage, err := json.Marshal(message)

			if err != nil {
				return err
			}

			// Adds the message to the currenct Subscriptions
			b.Proxy.Subscriptions = append(b.Proxy.Subscriptions, marshalledMessage)
			b.Proxy.MessageChannel <- marshalledMessage
		}
	}

	return nil
}

// Parses the response received to a JSON struct
func MakeResponse(b []byte) (interface{}, error) {
	byteString := string(b)
	byteString = byteString[1:]
	byteString = byteString[:len(byteString)-1]
	response := strings.Split(byteString, ",")

	if response[1] == "hb" || len(response) != 11 {
		return nil, nil
	}

	responseFloats := []float64{}

	for _, resp := range response {
		respFloats, err := strconv.ParseFloat(resp, 64)

		if err != nil {
			return nil, err
		}

		responseFloats = append(responseFloats, respFloats)
	}

	result := &TickerResponse{
		ChannelId:       int(responseFloats[0]),
		Bid:             responseFloats[1],
		BidSize:         responseFloats[2],
		Ask:             responseFloats[3],
		AskSize:         responseFloats[4],
		DailyChange:     responseFloats[5],
		DailyChangePrec: responseFloats[6],
		LastPrice:       responseFloats[7],
		Volume:          responseFloats[8],
		High:            responseFloats[9],
		Low:             responseFloats[10],
	}

	return result, nil
}

// Returns false if at least one of these condition is verified:
// 	- The Bitfinex structure has not been initialized
// 	- The Kafka channel has not been initialized
// 	- The Proxy has not been initialized or is nil
func (b *Bitfinex) IsClean() error {
	if reflect.DeepEqual(b, &Bitfinex{}) {
		return errors.New("[Bitfinex] Bitfinex structure cannot be nil\n")
	}

	if b.KafkaChannel == nil {
		return errors.New("[Bitfinex] Bitfinex structure doesn't have any kafka channel.")
	}

	if err := b.Proxy.IsClean(); err != nil {
		return err
	}

	return nil
}

// Translates a CurrencySlice (which contains CurrencyPair) to an array of strings.
// Adapts each string the specificities of each platform
// (ex: GDAX = "BTC-USD", Bitfinex = "tBTCUSD", ...)
func (b *Bitfinex) TranslateCurrency(c currency.CurrencySlice) []string {
	result := []string{}

	for _, cp := range c {
		result = append(result, cp.ToBitfinex())
	}

	return result
}
