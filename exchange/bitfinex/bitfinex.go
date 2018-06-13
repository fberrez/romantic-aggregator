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

	Subscribe   string = "subscribe"
	Unsubscribe string = "unsubscribe"

	Subscribed   string = "subscribed"
	Unsubscribed string = "unsubscribed"
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
			parsedResponse, err := b.makeResponse(response)

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
func (b *Bitfinex) NewMessage(isSubscribe bool, symbols []string, channels []string) error {
	for _, symbol := range symbols {
		for _, channel := range channels {
			var message interface{}

			if isSubscribe {
				message = Message{
					Event:   Subscribe,
					Symbol:  symbol,
					Channel: channel,
				}
			} else {
				// If it the message is a unsubscribe one,
				// we need to get the id of chan we want to remove
				chanId, err := b.getChanId(channel, symbol)

				if err != nil {
					return err
				}

				message = UnsubscribeMessage{
					Event:  Unsubscribe,
					ChanId: chanId,
				}
			}

			messageByte, err := json.Marshal(message)

			if err != nil {
				return fmt.Errorf("[Bitfinex] An error occured while trying to send a new message:\n\t- message: %v\n\t- error: %v", message, err)
			}

			b.Proxy.MessageChannel <- messageByte
		}
	}

	return nil
}

// Sends every messages to the Message Channel.
// These messages will be send by the proxy to the webocket
func (b *Bitfinex) sendMessage(message Message) error {
	marshalledMessage, err := json.Marshal(message)

	if err != nil {
		return err
	}

	b.Proxy.MessageChannel <- marshalledMessage

	return nil
}

// Parses the response received to a JSON struct
func (b *Bitfinex) makeResponse(data []byte) (interface{}, error) {
	byteString := string(data)

	switch byteString[0] {
	// a '[' is the first character of a ticker response
	case '[':
		return makeTickerResponse(data)
		// a '{' is the first character of a (un)subscribe reponse
	case '{':
		return b.manageSubscriptions(data)
	}

	return nil, fmt.Errorf("[Bitfinex] Cannot understand this following answer: %v", string(data))
}

func makeTickerResponse(data []byte) (interface{}, error) {
	byteString := string(data)
	// Removes the first and the last character of the response which are "[]"
	byteString = byteString[1:]
	byteString = byteString[:len(byteString)-1]

	// Splits the string
	response := strings.Split(byteString, ",")
	// If the first element of the array is "hb",
	// it means that there is nothing new
	if response[1] == "hb" || len(response) != 11 {
		return nil, nil
	}

	// Array which will contain the strings which were in the splitted array,
	// converted to float64
	responseFloats := []float64{}

	// Converts each strings to float64
	for _, resp := range response {
		respFloats, err := strconv.ParseFloat(resp, 64)

		if err != nil {
			return nil, err
		}

		responseFloats = append(responseFloats, respFloats)
	}

	// Builds the Ticker Response
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

// Manages subscriptions, whether subscribing or unsubscribing
func (b *Bitfinex) manageSubscriptions(data []byte) (interface{}, error) {
	dataJSON := &struct {
		Event string
	}{}
	err := json.Unmarshal(data, dataJSON)

	if err != nil {
		return nil, err
	}

	// Determines the event of the response ("subscribed" or "unsubscribed")
	switch dataJSON.Event {
	case Subscribed:
		response := SubscribeResponse{}
		if err = json.Unmarshal(data, &response); err != nil {
			return nil, err
		}

		b.manageSubscribe(response)
	case Unsubscribed:
		response := UnsubscribeResponse{}

		if err = json.Unmarshal(data, &response); err != nil {
			return nil, err
		}

		b.manageUnsubscribe(response)
	}

	log.Printf("[Bitfinex] Current Subscriptions: %v\n", b.Subscriptions)

	return nil, nil
}

// Adds the new subscription to the list of current subscriptions
// and updates the subscriptions in the proxy side
func (b *Bitfinex) manageSubscribe(subscribeResponse SubscribeResponse) error {
	b.Subscriptions = append(b.Subscriptions, subscribeResponse)
	return b.updateSubscriptions()
}

// Remove the subscription, determined by the unsubscribe response,
// from the list of current subscriptions.
// Updates the subscriptions in the proxy side
func (b *Bitfinex) manageUnsubscribe(unsubscribeResponse UnsubscribeResponse) error {
	for i, sub := range b.Subscriptions {
		if sub.ChanId == unsubscribeResponse.ChanId {
			b.Subscriptions = append(b.Subscriptions[:i], b.Subscriptions[i+1:]...)
			break
		}
	}

	return b.updateSubscriptions()
}

// Returns the channel id of the subscription
// which manage the channel and the pair which are in arguments
// Returns -1 if the channel id cannot be found
func (b *Bitfinex) getChanId(channel string, pair string) (int, error) {
	for _, sub := range b.Subscriptions {
		if sub.Channel == channel && sub.Pair == pair {
			return sub.ChanId, nil
		}
	}

	return -1, fmt.Errorf("[Bitfinex] Cannot find ChanId:\n\t- (channel, pair): (%v, %v)\n\t- Current Subscriptions: %v", channel, pair, b.Subscriptions)
}

// Adds or removes subscriptions on the proxy side
func (b *Bitfinex) updateSubscriptions() error {
	b.Proxy.Subscriptions = [][]byte{}
	// Go throught the list
	for _, sub := range b.Subscriptions {
		var event string

		// Determines the event of the new message
		switch sub.Event {
		case Subscribed:
			event = Subscribe
		case Unsubscribed:
			event = Unsubscribed
		}

		newSub := &Message{
			Event:   event,
			Channel: sub.Channel,
			Symbol:  sub.Pair,
		}

		// Converts the message struct to []data
		newSubByte, err := json.Marshal(newSub)
		if err != nil {
			return err
		}

		b.Proxy.Subscriptions = append(b.Proxy.Subscriptions, newSubByte)
	}

	return nil
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
