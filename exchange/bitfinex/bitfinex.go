package bitfinex

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/fberrez/romantic-aggregator/aggregator"
	"github.com/fberrez/romantic-aggregator/currency"
	"github.com/fberrez/romantic-aggregator/websocket"
	"github.com/juju/errors"
	"github.com/sirupsen/logrus"
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
	uri url.URL       = url.URL{Scheme: "wss", Host: "api.bitfinex.com", Path: "/ws"}
	log *logrus.Entry = logrus.WithFields(logrus.Fields{"element": "exchange", "label": "Bitfinex"})
)

// Initializes the Bitfinex struct
func (b *Bitfinex) Initialize(aggregatorChan chan aggregator.SimpleTicker) error {
	log.Infof("Initializing")

	b.AggregatorChannel = aggregatorChan
	b.Proxy = &websocket.Proxy{
		Label: "Bitfinex",
	}

	b.InterruptChannel = make(chan bool)

	return b.Proxy.Initialize(uri)
}

// Starts the goroutine Listen and the loop
// which will send messages present in the queue
// and will wait for the SIGINT
func (b *Bitfinex) Start() error {
	log.Infof("Start in progress...")
	err := b.IsClean()

	// If the Bitfinex is not clean (see (*Bitfinex)IsClean definition),
	// the process returns an error
	if err != nil {
		return errors.Annotate(err, "bitfinex struct must be correctly initialized")
	}

	go b.ListenResponse()
	b.Proxy.Start()
	return nil
}

// Listens the Bitfinex's websocket and processes datas he receives
// before to send it to a kafka producer
func (b *Bitfinex) ListenResponse() {

	for {
		select {
		case response := <-b.Proxy.ResponseChannel:
			_, err := b.makeResponse(response)

			if err != nil {
				log.WithFields(logrus.Fields{
					"action": "listening to the responses sent by websocket",
				}).Errorf("%v", err)
				break
			}

		case interrupt := <-b.InterruptChannel:
			if interrupt {
				return
			}
		}
	}
}

// Builds a new message to send and adds it to the queue
func (b *Bitfinex) NewMessage(isSubscription bool, symbols []string, channels []string) error {
	for _, symbol := range symbols {
		for _, channel := range channels {
			var message interface{}

			if isSubscription {
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
					return errors.Annotate(err, "while trying to send a new message to websocket")
				}

				message = UnsubscribeMessage{
					Event:  Unsubscribe,
					ChanId: chanId,
				}
			}

			// Sends the message to the websocket client
			messageByte, err := json.Marshal(message)

			if err != nil {
				return errors.Annotatef(err, "message %v", message)
			}

			log.WithFields(logrus.Fields{"message": message}).Debugf("Sending new message to websocket")
			b.Proxy.MessageChannel <- messageByte
		}
	}

	return nil
}

// Parses the response received to a JSON struct
func (b *Bitfinex) makeResponse(data []byte) (interface{}, error) {
	byteString := string(data)

	switch byteString[0] {
	// a '[' is the first character of a ticker response
	case '[':
		return b.makeTickerResponse(data)
		// a '{' is the first character of a (un)subscribe response
	case '{':
		return b.manageSubscriptions(data)
	}

	return nil, errors.NotSupportedf("cannot understand the following answer: %v", string(data))
}

// Builds the response sent by the websocket to the client
func (b *Bitfinex) makeTickerResponse(data []byte) (*TickerResponse, error) {
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
			return nil, errors.NotSupportedf("cannot parse %v to float64: %v", resp, err)
		}

		responseFloats = append(responseFloats, respFloats)
	}

	// Builds the Ticker Response
	tickerResponse := &TickerResponse{
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

	if _, err := b.parseAndSendTickerResponseToAggregator(tickerResponse); err != nil {
		return nil, err
	}

	return tickerResponse, nil
}

// Parses and send a new ticker response to aggregator
func (b *Bitfinex) parseAndSendTickerResponseToAggregator(t *TickerResponse) (*aggregator.SimpleTicker, error) {
	symbol, err := b.getSymbol(t.ChannelId)

	if err != nil {
		return nil, errors.Annotate(err, "tried to send a ticker response to aggregator")
	}

	aggregatorTicker := &aggregator.SimpleTicker{
		Exchange: "Bitfinex",
		Symbol:   symbol,
		Price:    t.LastPrice,
		Bid:      t.Bid,
		Ask:      t.Ask,
		Volume:   t.Volume,
	}

	b.AggregatorChannel <- *aggregatorTicker

	return aggregatorTicker, nil
}

// Manages subscriptions, whether subscribing or unsubscribing
func (b *Bitfinex) manageSubscriptions(data []byte) (interface{}, error) {
	dataJSON := &struct {
		Event string
	}{}
	err := json.Unmarshal(data, dataJSON)

	if err != nil {
		return nil, errors.Annotatef(err, "tried to parse a []byte %v to JSON", string(data))
	}

	// Determines the event of the response ("subscribed" or "unsubscribed")
	switch dataJSON.Event {
	case Subscribed:
		response := SubscribeResponse{}
		if err = json.Unmarshal(data, &response); err != nil {
			return nil, errors.Annotatef(err, "tried to unmarshal a subscribe response %v", string(data))
		}

		b.manageSubscribe(response)
	case Unsubscribed:
		response := UnsubscribeResponse{}

		if err = json.Unmarshal(data, &response); err != nil {
			return nil, errors.Annotatef(err, "tried to unmarshal an unsubscribe response %v", string(data))
		}

		b.manageUnsubscribe(response)
	}

	log.WithFields(logrus.Fields{"subscriptions": b.Subscriptions}).Debugf("Current Subscriptions")

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

	return -1, errors.NotFoundf("cannot find chanId:\n\t- (channel, pair): (%v, %v)\n\t- Current subscriptions: %v", channel, pair, b.Subscriptions)
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
			return errors.Annotatef(err, "tried to marshal new subscribe message %v", newSub)
		}

		b.Proxy.Subscriptions = append(b.Proxy.Subscriptions, newSubByte)
	}

	return nil
}

// Returns symbol handled by the channel id
func (b *Bitfinex) getSymbol(channelId int) (string, error) {
	for _, subscription := range b.Subscriptions {
		if subscription.ChanId == channelId {
			return subscription.Pair, nil
		}
	}

	return "", errors.NotFoundf("channel ID (%d) not found in the current subscriptions. Cannot find any symbol.", channelId)
}

// Returns false if at least one of these condition is verified:
// 	- The Bitfinex structure has not been initialized
// 	- The Kafka channel has not been initialized
// 	- The Proxy has not been initialized or is nil
func (b *Bitfinex) IsClean() error {
	if reflect.DeepEqual(b, &Bitfinex{}) {
		return errors.NotAssignedf("bitfinex structure cannot be nil")
	}

	if b.AggregatorChannel == nil {
		return errors.NotAssignedf("bitfinex structure doesn't have any exchange channel.")
	}

	if err := b.Proxy.IsClean(); err != nil {
		return err
	}

	return nil
}

// Translates a CurrencySlice (which contains CurrencyPair) to an array of strings.
// Adapts each string the specificities of each platform
// (ex: GDAX = "BTC-USD", Bitfinex = "tBTCUSD", ...)
func (b *Bitfinex) TranslateCurrency(c currency.CurrencySlice) ([]string, error) {
	result := []string{}

	for _, cp := range c {
		formattedCurrencyPair, err := cp.ToBitfinex()

		if err != nil {
			return result, err
		}

		result = append(result, formattedCurrencyPair)
	}

	return result, nil
}

// Handles SIGINT
func (b *Bitfinex) Interrupt() {
	log.Debug("Closing Bitfinex")
	b.Proxy.Interrupt()
	b.InterruptChannel <- true
}
