package gdax

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/fberrez/romantic-aggregator/aggregator"
	"github.com/fberrez/romantic-aggregator/currency"
	"github.com/fberrez/romantic-aggregator/websocket"
	"github.com/juju/errors"
	"github.com/sirupsen/logrus"
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
	uri url.URL       = url.URL{Scheme: "wss", Host: "ws-feed.pro.coinbase.com", Path: "/"}
	log *logrus.Entry = logrus.WithFields(logrus.Fields{"element": "exchange", "label": "GDAX"})
)

// Initializes the GDAX struct
func (g *GDAX) Initialize(aggregatorChan chan aggregator.SimpleTicker) error {
	log.Infof("Initializing")
	g.AggregatorChannel = aggregatorChan
	g.Proxy = &websocket.Proxy{
		Label: "GDAX",
	}
	g.InterruptChannel = make(chan bool)

	return g.Proxy.Initialize(uri)
}

// Starts the goroutine Listen and the loop
// which will send messages present in the queue
// and will wait for the SIGINT
func (g *GDAX) Start() error {
	log.Infof("Start in progress...")
	err := g.isClean()

	// If the GDAX is not clean (see (*GDAX)IsClean definition),
	// the process returns an error
	if err != nil {
		return errors.Annotate(err, "gdax struct must be correctly initialized")
	}

	go g.ListenResponse()
	g.Proxy.Start()

	return nil
}

// Receives response sent by the proxy in the response_channel.
// Processes each of them before to send them in the kafka channel.
func (g *GDAX) ListenResponse() {
	for {
		select {
		case response := <-g.Proxy.ResponseChannel:
			_, err := g.makeResponse(response)

			if err != nil {
				log.WithFields(logrus.Fields{
					"action": "listening to the responses sent by websocket",
				}).Errorf("%v", err)
				break
			}

		case interrupt := <-g.InterruptChannel:
			if interrupt {
				return
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
		return errors.Annotate(err, "while trying to send a new message to websocket")
	}

	return nil
}

// Sends every messages to the Message Channel.
// These messages will be send by the proxy to the webocket
func (g *GDAX) sendMessage(message Message) error {
	log.WithFields(logrus.Fields{"message": message}).Debugf("Sending new message to websocket")
	marshalledMessage, err := json.Marshal(message)

	if err != nil {
		return errors.Annotatef(err, "tried to send message %v", message)
	}

	g.Proxy.MessageChannel <- marshalledMessage

	return nil
}

// Parses the response received to a JSON struct
func (g *GDAX) makeResponse(b []byte) (interface{}, error) {
	response := &Response{}
	err := json.Unmarshal(b, &response)

	if err != nil {
		return nil, errors.Annotatef(err, "tried to make response %v", string(b))
	}

	// If it is a subscriptions feedback
	if response.Type == "subscriptions" {
		return g.manageSubscriptions(b)
	}

	if response.Type == "ticker" {
		tickerResponse := &TickerResponse{}
		err = json.Unmarshal(b, &tickerResponse)

		if err != nil {
			return nil, errors.Annotatef(err, "tried to make a new ticker response %v", string(b))
		}

		return g.parseAndSendTickerResponseToAggregator(tickerResponse)
	}

	return nil, nil
}

// Parses and send a new ticker response to aggregator
func (g *GDAX) parseAndSendTickerResponseToAggregator(t *TickerResponse) (*aggregator.SimpleTicker, error) {
	price, err := strconv.ParseFloat(t.Price, 64)
	bid, err := strconv.ParseFloat(t.BestBid, 64)
	ask, err := strconv.ParseFloat(t.BestAsk, 64)

	if err != nil {
		return nil, errors.Annotatef(err, "tried make an new ticker response %v", t)
	}

	volume, err := getVolume(t.ProductId)

	if err != nil || volume == 0.0 {
		return nil, err
	}

	aggregatorTicker := &aggregator.SimpleTicker{
		Exchange: "GDAX",
		Symbol:   fmt.Sprintf("%s%s", t.ProductId[:3], t.ProductId[4:]),
		Price:    price,
		Bid:      bid,
		Ask:      ask,
		Volume:   volume,
	}

	g.AggregatorChannel <- *aggregatorTicker

	return aggregatorTicker, nil
}

// Returns the volume for a specific currency pair
func getVolume(symbol string) (float64, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := client.Get(fmt.Sprintf("https://api.pro.coinbase.com/products/%s/ticker", symbol))

	if err != nil {
		return 0.0, errors.Annotatef(err, "tried to get volume of %s", symbol)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return 0.0, errors.Annotatef(err, "cannot read the response given by the gdax api about %s", symbol)
	}

	tickerApiResponse := &TickerApiResponse{}
	err = json.Unmarshal(body, tickerApiResponse)

	if err != nil {
		return 0.0, errors.Annotatef(err, "tried to unmarshal the response given by the gdax api about %s", symbol)
	}

	volume, err := strconv.ParseFloat(tickerApiResponse.Volume, 64)

	if err != nil {
		return 0.0, errors.Annotatef(err, "tried to parse the volume")
	}

	return volume, nil
}

// Processes the subscription feedback sent by the websocket to the proxy
// Builds a new ready-to-be-sent message which contains every current subscriptions.
// Usefull when the websocket is closed.
func (g *GDAX) manageSubscriptions(b []byte) (interface{}, error) {
	subscriptionResponse := &SubscriptionResponse{}
	err := json.Unmarshal(b, &subscriptionResponse)

	if err != nil {
		return nil, errors.Annotatef(err, "tried to get infos about a new subscription response")
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
	g.Proxy.Subscriptions = [][]byte{subscriptionMessageByte}

	log.WithFields(logrus.Fields{"subscriptions": *g.Subscriptions}).Debug("Current Subscriptions")

	return subscriptionMessage, nil
}

// Returns false if at least one of these condition is verified:
// 	- The GDAX structure has not been initialized
// 	- The Kafka channel has not been initialized
// 	- The Proxy has not been initialized or is nil
func (g *GDAX) isClean() error {
	if reflect.DeepEqual(g, &GDAX{}) {
		return errors.NotAssignedf("gdax structure cannot be nil")
	}

	if g.AggregatorChannel == nil {
		return errors.NotAssignedf("gdax structure doesn't have any exchange channel")
	}

	if err := g.Proxy.IsClean(); err != nil {
		return err
	}

	return nil
}

// Translates a CurrencySlice (which contains CurrencyPair) to an array of strings.
// Adapts each string the specificities of each platform
// (ex: GDAX = "BTC-USD", Bitfinex = "tBTCUSD", ...)
func (g *GDAX) TranslateCurrency(c currency.CurrencySlice) ([]string, error) {
	result := []string{}

	for _, cp := range c {
		formattedCurrencyPair, err := cp.ToGDAX()

		if err != nil {
			return result, err
		}

		result = append(result, formattedCurrencyPair)
	}

	return result, nil
}

// Handles SIGINT
func (g *GDAX) Interrupt() {
	log.Debug("Closing GDAX")
	g.Proxy.Interrupt()
	g.InterruptChannel <- true
}
