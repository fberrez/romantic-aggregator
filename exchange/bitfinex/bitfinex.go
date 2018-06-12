package bitfinex

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"

	"github.com/fberrez/romantic-aggregator/currency"
	"github.com/gorilla/websocket"
	"github.com/op/go-logging"
)

const (
	Book   string = "book"
	Trade  string = "trades"
	Ticker string = "ticker"
)

var (
	uri url.URL = url.URL{Scheme: "wss", Host: "api.bitfinex.com", Path: "/ws"}
	log         = logging.MustGetLogger("bitfinex")
)

// Initializes the Bitfinex struct
func (b *Bitfinex) Initialize(kafkaChannel chan interface{}) error {
	c, _, err := websocket.DefaultDialer.Dial(uri.String(), nil)

	if err != nil {
		return err
	}

	b.WssUrl = uri
	b.Conn = c
	b.MessageQueue = []Message{}
	b.KafkaChannel = kafkaChannel

	return nil
}

// Starts the goroutine Listen and the loop
// which will send messages present in the queue
// and will wait for the SIGINT
func (b *Bitfinex) Start(testing bool) error {
	//Defines the SIGINT
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	err := b.IsClean()

	// If the Bitfinex is not clean (see (*Bitfinex)IsClean definition),
	// the process returns an error
	if err != nil {
		return errors.New(fmt.Sprintf("Bitfinex structure needs to be properly initialized before to continue.\n\tErr: %v", err))
	}

	done := make(chan struct{})
	log.Infof("Connecting to %s", b.WssUrl.String())

	// Listen and process every datas
	// received by the connection to the websocket
	go b.Listen(done)

	// In case of testing, a SIGINT is send to the `interrupt` channel
	// to stop this process
	if testing {
		interrupt <- os.Interrupt
	}

	for {
		if len(b.MessageQueue) > 0 {
			if err := b.SendMessage(); len(err) > 0 {
				log.Error("Error occured while sending message(s): ", err)
			}
		}

		select {
		case <-interrupt:
			b.Interrupt()
			return nil
		}
	}
}

// Listens the Bitfinex's websocket and processes datas he receives
// before to send it to a kafka producer
func (b *Bitfinex) Listen(done chan struct{}) {
	log.Info("Listenning Bitfinex at %s", b.WssUrl.String())
	defer close(done)

	for {
		_, message, err := b.Conn.ReadMessage()
		if err != nil {
			log.Error("[Bitfinex] Listen:", err)
			return
		}

		response, err := MakeResponse(message)

		if err != nil {
			log.Error("MakeResponse:", err)
			return
		}

		if response != nil {
			log.Infof("Sending Message %v to Kafka from Bitfinex", response)
			b.KafkaChannel <- response
		}
	}
}

// Builds a new message to send and adds it to the queue
func (b *Bitfinex) NewMessage(aEvent string, symbols []string, channels []string) {
	for _, symbol := range symbols {
		for _, channel := range channels {
			message := Message{
				Event:   aEvent,
				Symbol:  symbol,
				Channel: channel,
			}

			b.MessageQueue = append(b.MessageQueue, message)
		}
	}
}

// Sends messages present in the queue
func (b *Bitfinex) SendMessage() []error {
	var errors []error
	// currentIndex is a counter which determines
	// the current index which is processed
	currentIndex := 0

	for _, message := range b.MessageQueue {
		log.Infof("Sending Message %v to Bitfinex", message)
		// Parses the message to JSON
		messageJSON, err := json.Marshal(message)

		// If an error occured, the current index counter is incremented
		// and the error is added to the arrays of errors.
		// Finally, the current loop is stopped and the next one is started
		if err != nil {
			currentIndex += 1
			errors = append(errors, err)
			continue
		}

		err = b.Conn.WriteMessage(websocket.TextMessage, []byte(messageJSON))

		if err != nil {
			currentIndex += 1
			errors = append(errors, err)
			continue
		}

		// If the sending of the message has been a success,
		// the message is removed from the message queue
		b.MessageQueue = append(b.MessageQueue[:currentIndex], b.MessageQueue[currentIndex+1:]...)
	}

	return errors
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

// Handles the SIGINT
func (b *Bitfinex) Interrupt() {
	b.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

// Returns false if at least one of these condition is verified:
// 	- The Bitfinex structure has not been initialized
// 	- The URL is not set or not correct
// 	- The message queue has not been initialized
// 	- The Kafka channel has not been initialized
// 	- The Connection with the Bitfinex websocket has not been initialized or is nil
func (b *Bitfinex) IsClean() error {
	if reflect.DeepEqual(b, &Bitfinex{}) {
		return errors.New("Bitfinex structure cannot be nil\n")
	}

	if (b.WssUrl == url.URL{}) || (b.WssUrl.String() != uri.String()) {
		return errors.New("Bitfinex structure doesn't have a good Websocket URL.")
	}

	if b.MessageQueue == nil {
		return errors.New("Bitfinex structure doesn't have a good Message Queue.")
	}

	if b.KafkaChannel == nil {
		return errors.New("Bitfinex structure doesn't have any kafka channel.")
	}

	if b.Conn == nil || reflect.DeepEqual(b.Conn, &websocket.Conn{}) {
		return errors.New("Bitfinex structure doesn't have any connection etablished with the websocket")
	}

	return nil
}

// Translates a CurrencySlice (which contains CurrencyPair) to an array of string.
// Adapts each string the specificities of each platform
// (ex: GDAX = "BTC-USD", BITFINEX = "tBTCUSD", ...)
func (b *Bitfinex) TranslateCurrency(c currency.CurrencySlice) []string {
	result := []string{}

	for _, cp := range c {
		result = append(result, cp.ToBitfinex())
	}

	return result
}
