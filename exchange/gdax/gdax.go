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

	Subscribe   string = "subscribe"
	Unsubscribe string = "unsubscribe"
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
	err := g.isClean()

	// If the GDAX is not clean (see (*GDAX)IsClean definition),
	// the process returns an error
	if err != nil {
		return errors.New(fmt.Sprintf("[GDAX] GDAX structure needs to be properly initialized before to continue.\n\tErr: %v", err))
	}

	go g.ListenResponse()
	g.Proxy.Start(testing)

	return nil
}

// Receives response sent by the proxy in the response_channel.
// Processes each of them before to send them in the kafka channel.
func (g *GDAX) ListenResponse() {
	for {
		select {
		case response := <-g.Proxy.ResponseChannel:
			parsedResponse, err := g.makeResponse(response)

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
func (g *GDAX) NewMessage(isSubscribe bool, productIds []string, channels []string) error {
	message := Message{
		ProductIds: productIds,
		Channels:   channels,
	}

	switch isSubscribe {
	case true:
		message.Type = Subscribe
	case false:
		message.Type = Unsubscribe
	}

	if err := g.sendMessage(message); err != nil {
		return fmt.Errorf("[GDAX] Error occured while trying to send a message\n\t- message:%v\n\t- err:%v", message, err)
	}

	return nil
}

// Sends every messages to the Message Channel.
// These messages will be send by the proxy to the webocket
func (g *GDAX) sendMessage(message Message) error {
	marshalledMessage, err := json.Marshal(message)

	if err != nil {
		return err
	}

	g.Proxy.MessageChannel <- marshalledMessage

	return nil
}

// Parses the response received to a JSON struct
func (g *GDAX) makeResponse(b []byte) (interface{}, error) {
	response := &Response{}
	err := json.Unmarshal(b, &response)

	if err != nil {
		return nil, err
	}

	var result interface{}

	// If it is a subscriptions feedback
	if response.Type == "subscriptions" {
		return g.manageSubscriptions(b)
	}

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

// Processes the subscription feedback sent by the websocket to the proxy
// Builds a new ready-to-be-sent message which contains every current subscriptions.
// Usefull when the websocket is closed.
func (g *GDAX) manageSubscriptions(b []byte) (interface{}, error) {
	subscriptionResponse := &SubscriptionResponse{}
	err := json.Unmarshal(b, &subscriptionResponse)

	if err != nil {
		return nil, err
	}

	subscriptionMessage := &Message{
		Type: Subscribe,
	}

	for _, channel := range subscriptionResponse.Channels {
		subscriptionMessage.Channels = append(subscriptionMessage.Channels, channel.Name)
		for _, productId := range channel.ProductIds {
			subscriptionMessage.ProductIds = append(subscriptionMessage.ProductIds, productId)
		}
	}

	subscriptionMessageByte, err := json.Marshal(subscriptionMessage)

	if err != nil {
		return nil, err
	}

	// Updates current subscriptions in the exchange side
	g.Subscriptions = subscriptionMessage
	// Updates current subscriptions in the proxy side
	g.Proxy.Subscriptions = append(g.Proxy.Subscriptions, subscriptionMessageByte)

	log.Printf("[GDAX] Current Subscriptions: %v\n", g.Subscriptions)

	return subscriptionMessage, nil
}

// Returns false if at least one of these condition is verified:
// 	- The GDAX structure has not been initialized
// 	- The Kafka channel has not been initialized
// 	- The Proxy has not been initialized or is nil
func (g *GDAX) isClean() error {
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
