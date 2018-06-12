package exchange

import (
	"log"
	"sync"

	"github.com/fberrez/romantic-aggregator/currency"
	"github.com/fberrez/romantic-aggregator/exchange/bitfinex"
	"github.com/fberrez/romantic-aggregator/exchange/gdax"
)

type Fetcher interface {
	// Initializes the Fetcher and his websocket dialer
	Initialize(chan interface{}) error

	// Launches the gorountine "Listen"
	// and waits for the sigterm
	// the boolean placed as a defined argument whether it is a test or not
	Start(bool) error

	// Translates a CurrencySlice (which contains CurrencyPair) to an array of strings.
	// Adapts each string the specificities of each platform
	// (ex: GDAX = "BTC-USD", Bitfinex = "tBTCUSD", ...)
	TranslateCurrency(currency.CurrencySlice) []string

	// Builds a new message to send to the fetcher websocket
	SendMessage(string, []string, []string) error
}

// FetcherGroup contains an array of Fetcher
// and a WaitGroup (which waits for a collection of goroutines to finish)
type FetcherGroup struct {
	fetchers     []Fetcher
	waitGroup    sync.WaitGroup
	kafkaChannel chan interface{}
}

var (
	Drivers = map[string]Fetcher{
		"GDAX":     &gdax.GDAX{},
		"Bitfinex": &bitfinex.Bitfinex{},
	}
)

// Initializes a FetcherGroup and
// each Fetcher which are in the FetcherGroup's fetchers
func Initialize(kafkaChan chan interface{}) *FetcherGroup {
	fg := &FetcherGroup{
		fetchers:     make([]Fetcher, 0),
		waitGroup:    sync.WaitGroup{},
		kafkaChannel: kafkaChan,
	}

	for driverName, driver := range Drivers {
		err := driver.Initialize(fg.kafkaChannel)

		if err != nil {
			log.Printf("Error occured while initializing %s: %v\n", driverName, err)
		} else {
			fg.fetchers = append(fg.fetchers, driver)
		}
	}

	return fg
}

// Starts eacher Fetcher which are in the FetcherGroup's fetchers
func (fg *FetcherGroup) Start() {
	for index, fetcher := range fg.fetchers {
		fg.waitGroup.Add(1)

		go func(fetcher Fetcher) {
			defer fg.waitGroup.Done()
			err := fetcher.Start(false)

			if err != nil {
				log.Printf("Error occured while trying to start #%v: %v\n", index, err)
			}
		}(fetcher)
	}

	fg.waitGroup.Wait()
}

// Sends message to each Fetcher's websocket
func (fg *FetcherGroup) SendMessage(aType string, productIds currency.CurrencySlice, channels []string) {
	for index, fetcher := range fg.fetchers {
		err := fetcher.SendMessage(aType, fetcher.TranslateCurrency(productIds), channels)

		if err != nil {
			log.Printf("Error occured while trying to send a message #%v: %v\n", index, err)
		}
	}
}
