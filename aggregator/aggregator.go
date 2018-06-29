package aggregator

import (
	"time"

	"github.com/sirupsen/logrus"
)

// Struct which contains informations about one specific channel & symbols
type SimpleTicker struct {
	// Name of the exchange (ex: Bitfinex)
	Exchange string `json:"exchange"`

	// Symbol of the currency pair (ex: BTCUSD)
	Symbol string `json:"symbol"`

	// Last price
	Price float64 `json:"price"`

	// Last bid
	Bid float64 `json:"bid"`

	// Last ask
	Ask float64 `json:"ask"`

	// Last volume
	Volume float64 `json:"volume"`
}

// Struct which contains informations about an aggregator
type Aggregator struct {
	// Array which contains averages of received tickers
	tickers []*Ticker

	//  Time of the last update
	lastUpdate time.Time

	// Channel which receives a ticker to add to the array of tickers
	AggregatorChannel chan SimpleTicker

	// Channel which handles the SIGINT
	interruptChannel chan bool

	// Channel which makes the relation
	// between the aggregator and the Kafka producer
	kafkaChannel chan interface{}

	// Timer which determines the interval between
	// each new messages sent to the Kafka producer
	timer *time.Ticker

	// Channel which handles timer updates
	intervalChannel chan Interval
}

// Struct which contains the average values ​​of each ticker received
type Ticker struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Bid    float64 `json:"bid"`
	Ask    float64 `json:"ask"`
	Volume float64 `json:"volume"`
}

// Interval between each messages sent to kafka producer
type Interval int

// Interval in seconds
const (
	OneMinute        Interval = 60
	ThreeMinutes     Interval = 180
	FiveMinutes      Interval = 300
	FifTeenMinutes   Interval = 900
	ThirtyMinutes    Interval = 1800
	FortyFiveMinutes Interval = 2700
	OneHour          Interval = 3600
	TwoHours         Interval = 7200
	ThreeHours       Interval = 10800
	FourHours        Interval = 14400
	OneDay           Interval = 86400
	OneWeek          Interval = 604800
	// One month = 30 days
	OneMonth Interval = 2592000
)

var (
	defaultInterval Interval      = OneMinute
	log             *logrus.Entry = logrus.WithFields(logrus.Fields{"element": "aggregator"})
)

// Initializes a new aggregator struct
func Initialize(kafkaChan chan interface{}) *Aggregator {
	aggregator := &Aggregator{
		intervalChannel:   make(chan Interval),
		tickers:           []*Ticker{},
		lastUpdate:        time.Now(),
		AggregatorChannel: make(chan SimpleTicker),
		interruptChannel:  make(chan bool),
		kafkaChannel:      kafkaChan,
		timer:             time.NewTicker(time.Duration(defaultInterval) * time.Second),
	}

	return aggregator
}

// Starts the loop which handles each signals:
// - Receiving a new ticker
// - A time interval completed
// - Updating timer
// - Closing/Stopping of the aggregator
func (a *Aggregator) Start() {
AggregatorLoop:
	for {
		select {
		// Ticker received
		case simpleTicker := <-a.AggregatorChannel:
			log.WithFields(logrus.Fields{"ticker": simpleTicker}).Debug("Ticker Received")
			a.makeAverage(simpleTicker)

		// Time interval completed
		case t := <-a.timer.C:
			for _, ticker := range a.tickers {
				log.WithField("ticker", *ticker).Infof("Send Ticker to Kafka at %v", t)
				a.kafkaChannel <- ticker
			}

		// Updates timer
		case interval := <-a.intervalChannel:
			a.timer = time.NewTicker(time.Duration(interval) * time.Second)

		// SIGINT received
		case signal := <-a.interruptChannel:
			if signal {
				log.Info("Closing aggregator")
				break AggregatorLoop
			}
		}
	}
}

// Modifies the timer interval
func (a *Aggregator) SetInterval(interval Interval) {
	log.WithField("interval", interval).Infof("The interval of the ticker has been changed (in sec)")
	a.intervalChannel <- interval
}

// Calculates the average of a ticker
func (a *Aggregator) makeAverage(t SimpleTicker) {
	currentTicker := a.findTicker(t)

	currentTicker.Price = (currentTicker.Price*currentTicker.Volume + t.Price*t.Volume) / (currentTicker.Volume + t.Volume)
	currentTicker.Bid = (currentTicker.Bid*currentTicker.Volume + t.Bid*t.Volume) / (currentTicker.Volume + t.Volume)
	currentTicker.Ask = (currentTicker.Ask*currentTicker.Volume + t.Ask*t.Volume) / (currentTicker.Volume + t.Volume)
	currentTicker.Volume = (currentTicker.Volume + t.Volume) / 2

	log.WithFields(logrus.Fields{"ticker": currentTicker}).Debug("Ticker Calculated")
}

// Finds or creates a new ticker to return
func (a *Aggregator) findTicker(t SimpleTicker) *Ticker {
	for _, ticker := range a.tickers {
		if ticker.Symbol == t.Symbol {
			return ticker
		}
	}

	newTicker := &Ticker{
		Symbol: t.Symbol,
		Price:  t.Price,
		Bid:    t.Bid,
		Ask:    t.Ask,
		Volume: t.Volume,
	}

	a.tickers = append(a.tickers, newTicker)

	return newTicker
}

// Stops the aggregator loop
func (a *Aggregator) Stop() {
	a.interruptChannel <- true
}
