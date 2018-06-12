package gdax

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"reflect"

	"github.com/fberrez/romantic-aggregator/currency"
	"github.com/fberrez/romantic-aggregator/websocket"
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
)

// Initializes the GDAX struct
func (g *GDAX) Initialize(kafkaChannel chan interface{}) error {
	log.Println("[GDAX] Initializing GDAX")
	g.KafkaChannel = kafkaChannel
	g.Proxy = &websocket.Proxy{
		Label: "GDAX",
	}

	return g.Proxy.Initialize(uri, kafkaChannel)
}

// Starts the goroutine Listen and the loop
// which will send messages present in the queue
// and will wait for the SIGINT
func (g *GDAX) Start(testing bool) error {
	err := g.IsClean()

	// If the GDAX is not clean (see (*GDAX)IsClean definition),
	// the process returns an error
	if err != nil {
		return errors.New(fmt.Sprintf("[GDAX] GDAX structure needs to be properly initialized before to continue.\n\tErr: %v", err))
	}

	go g.ListenResponse()
	g.Proxy.Start(testing)

	return nil
}

func (g *GDAX) ListenResponse() {
	for {
		select {
		case response := <-g.Proxy.ResponseChannel:
			parsedResponse, err := MakeResponse(response)

			if err != nil {
				log.Printf("[GDAX] Error occured while listening GDAX Websocket: %v", err)
				break
			}

			if parsedResponse != nil {
				log.Printf("[GDAX] Sending Message %v to Kafka from GDAX", parsedResponse)
				g.KafkaChannel <- parsedResponse
			}
		}
	}
}

// Builds a new message to send and adds it to the queue
func (g *GDAX) SendMessage(aType string, productIds []string, channels []string) error {
	message := Message{
		Type:       aType,
		ProductIds: productIds,
		Channels:   channels,
	}

	marshalledMessage, err := json.Marshal(message)

	if err != nil {
		return err
	}

	// Adds the message to the currenct Subscriptions
	g.Proxy.Subscriptions = append(g.Proxy.Subscriptions, marshalledMessage)
	g.Proxy.MessageChannel <- marshalledMessage

	return nil
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
	default:
		return nil, nil
	}

	err = json.Unmarshal(b, &result)

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Returns false if at least one of these condition is verified:
// 	- The GDAX structure has not been initialized
// 	- The Kafka channel has not been initialized
// 	- The Proxy has not been initialized or is nil
func (g *GDAX) IsClean() error {
	if reflect.DeepEqual(g, &GDAX{}) {
		return errors.New("[GDAX] GDAX structure cannot be nil\n")
	}

	if g.KafkaChannel == nil {
		return errors.New("[GDAX] GDAX structure doesn't have any kafka channel.")
	}

	if err := g.Proxy.IsClean(); err != nil {
		return err
	}

	return nil
}

// Translates a CurrencySlice (which contains CurrencyPair) to an array of strings.
// Adapts each string the specificities of each platform
// (ex: GDAX = "BTC-USD", Bitfinex = "tBTCUSD", ...)
func (g *GDAX) TranslateCurrency(c currency.CurrencySlice) []string {
	result := []string{}

	for _, cp := range c {
		result = append(result, cp.ToGDAX())
	}

	return result
}
