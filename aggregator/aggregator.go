package aggregator

import (
	"time"

	"github.com/sirupsen/logrus"
)

type SimpleTicker struct {
	Exchange string  `json:"exchange"`
	Symbol   string  `json:"symbol"`
	Price    float64 `json:"price"`
	Bid      float64 `json:"bid"`
	Ask      float64 `json:"ask"`
	Volume   float64 `json:"volume"`
}

type Aggregator struct {
	tickers           []*Ticker
	lastUpdate        time.Time
	AggregatorChannel chan SimpleTicker
	interruptChannel  chan bool
	kafkaChannel      chan interface{}
}

type Ticker struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Bid    float64 `json:"bid"`
	Ask    float64 `json:"ask"`
	Volume float64 `json:"volume"`
}

var (
	log *logrus.Entry = logrus.WithFields(logrus.Fields{"element": "aggregator"})
)

func Initialize(kafkaChan chan interface{}) *Aggregator {
	aggregator := &Aggregator{
		tickers:           []*Ticker{},
		lastUpdate:        time.Now(),
		AggregatorChannel: make(chan SimpleTicker),
		interruptChannel:  make(chan bool),
		kafkaChannel:      kafkaChan,
	}

	return aggregator
}

func (a *Aggregator) Start() {
AggregatorLoop:
	for {
		select {
		case simpleTicker := <-a.AggregatorChannel:
			log.WithFields(logrus.Fields{"ticker": simpleTicker}).Debug("Ticker Received")
			a.makeAverage(simpleTicker)

			for _, ticker := range a.tickers {
				a.kafkaChannel <- ticker
			}

		case signal := <-a.interruptChannel:
			if signal {
				log.Info("Closing aggregator")
				break AggregatorLoop
			}
		}
	}
}

func (a *Aggregator) makeAverage(t SimpleTicker) {
	currentTicker := a.findTicker(t)

	currentTicker.Price = (currentTicker.Price*currentTicker.Volume + t.Price*t.Volume) / (currentTicker.Volume + t.Volume)
	currentTicker.Bid = (currentTicker.Bid*currentTicker.Volume + t.Bid*t.Volume) / (currentTicker.Volume + t.Volume)
	currentTicker.Ask = (currentTicker.Ask*currentTicker.Volume + t.Ask*t.Volume) / (currentTicker.Volume + t.Volume)
	currentTicker.Volume = (currentTicker.Volume + t.Volume) / 2

	log.WithFields(logrus.Fields{"ticker": currentTicker}).Debug("Ticker Calculated")
}

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

func (a *Aggregator) Stop() {
	a.interruptChannel <- true
}
