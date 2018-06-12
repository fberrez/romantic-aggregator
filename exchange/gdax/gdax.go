package gdax

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"reflect"

	"github.com/fberrez/romantic-aggregator/currency"
	"github.com/gorilla/websocket"
	"github.com/op/go-logging"
)

// Defines the names of the channels present on GDAX
const (
	Heartbeat string = "heartbeat"
	Ticker    string = "ticker"
	Level2    string = "level2"
	Full      string = "full"
)

var (
	uri url.URL = url.URL{Scheme: "wss", Host: "ws-feed.gdax.com", Path: "/"}
	log         = logging.MustGetLogger("gdax")
)

// Initializes the GDAX struct
func (g *GDAX) Initialize(kafkaChannel chan interface{}) error {
	c, _, err := websocket.DefaultDialer.Dial(uri.String(), nil)

	if err != nil {
		return err
	}

	g.WssUrl = uri
	g.Conn = c
	g.MessageQueue = []Message{}
	g.KafkaChannel = kafkaChannel

	return nil
}

// Starts the goroutine Listen and the loop
// which will send messages present in the queue
// and will wait for the SIGINT
func (g *GDAX) Start(testing bool) error {
	//Defines the SIGINT
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	err := g.IsClean()

	// If the GDAX is not clean (see (*GDAX)IsClean definition),
	// the process returns an error
	if err != nil {
		return errors.New(fmt.Sprintf("GDAX structure needs to be properly initialized before to continue.\n\tErr: %v", err))
	}

	done := make(chan struct{})
	log.Infof("Connecting to %s", g.WssUrl.String())

	// Listen and process every datas
	// received by the connection to the websocket
	go g.Listen(done)

	// In case of testing, a SIGINT is send to the `interrupt` channel
	// to stop this process
	if testing {
		interrupt <- os.Interrupt
	}

	for {
		if len(g.MessageQueue) > 0 {
			if err := g.SendMessage(); len(err) > 0 {
				log.Error("Error occured while sending message(s): ", err)
			}
		}

		select {
		case <-interrupt:
			g.Interrupt()
			return nil
		}
	}
}

// Listens the GDAX's websocket and processes datas he receives
// before to send it to a kafka producer
func (g *GDAX) Listen(done chan struct{}) {
	log.Info("Listenning GDAX at %s", g.WssUrl.String())
	defer close(done)

	for {
		_, message, err := g.Conn.ReadMessage()
		if err != nil {
			log.Error("[GDAX] Listen:", err, "\n message:", message)
			return
		}

		response, err := MakeResponse(message)

		if err != nil {
			log.Error("MakeResponse:", err)
			return
		}

		if response != nil {
			log.Infof("Sending Message %v to Kafka from GDAX", response)
			g.KafkaChannel <- response
		}
	}
}

// Builds a new message to send and adds it to the queue
func (g *GDAX) NewMessage(aType string, productIds []string, channels []string) {
	message := Message{
		Type:       aType,
		ProductIds: productIds,
		Channels:   channels,
	}

	g.MessageQueue = append(g.MessageQueue, message)
}

// Sends messages present in the queue
func (g *GDAX) SendMessage() []error {
	var errors []error
	// currentIndex is a counter which determines
	// the current index which is processed
	currentIndex := 0

	for _, message := range g.MessageQueue {
		log.Infof("Sending Message %v to Gdax", message)
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

		err = g.Conn.WriteMessage(websocket.TextMessage, []byte(messageJSON))

		if err != nil {
			currentIndex += 1
			errors = append(errors, err)
			continue
		}

		// If the sending of the message has been a success,
		// the message is removed from the message queue
		g.MessageQueue = append(g.MessageQueue[:currentIndex], g.MessageQueue[currentIndex+1:]...)
	}

	return errors
}

// Parses the response received to a JSON struct
func MakeResponse(b []byte) (interface{}, error) {
	response := &Response{}
	err := json.Unmarshal(b, &response)

	if err != nil {
		return nil, err
	}

	if response.Type == "subscriptions" || response.Type == "unsubscribe" {
		return nil, nil
	}

	var result interface{}

	switch response.Type {
	case "ticker":
		result = &TickerResponse{}
	case "l2update":
		result = &L2UpdateResponse{}
	case "heartbeat":
		result = &HeartbeatResponse{}
	case "snapshot":
		result = &SnapshotResponse{}
	}

	err = json.Unmarshal(b, &result)

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Handles the SIGINT
func (g *GDAX) Interrupt() {
	g.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

// Returns false if at least one of these condition is verified:
// 	- The GDAX structure has not been initialized
// 	- The URL is not set or not correct
// 	- The message queue has not been initialized
// 	- The Kafka channel has not been initialized
// 	- The Connection with the GDAX websocket has not been initialized or is nil
func (g *GDAX) IsClean() error {
	if reflect.DeepEqual(g, &GDAX{}) {
		return errors.New("GDAX structure cannot be nil\n")
	}

	if (g.WssUrl == url.URL{}) || (g.WssUrl.String() != uri.String()) {
		return errors.New("GDAX structure doesn't have a good Websocket URL.")
	}

	if g.MessageQueue == nil {
		return errors.New("GDAX structure doesn't have a good Message Queue.")
	}

	if g.KafkaChannel == nil {
		return errors.New("GDAX structure doesn't have any kafka channel.")
	}

	if g.Conn == nil || reflect.DeepEqual(g.Conn, &websocket.Conn{}) {
		return errors.New("GDAX structure doesn't have any connection etablished with the websocket")
	}

	return nil
}

// Translates a CurrencySlice (which contains CurrencyPair) to an array of string.
// Adapts each string the specificities of each platform
// (ex: GDAX = "BTC-USD", BITFINEX = "tBTCUSD", ...)
func (g *GDAX) TranslateCurrency(c currency.CurrencySlice) []string {
	result := []string{}

	for _, cp := range c {
		result = append(result, cp.ToGDAX())
	}

	return result
}
