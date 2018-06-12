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

	// It is a loop which receives
	// and processes the datas (Unmarshal to a struct)
	Listen(chan struct{})

	// Launches the gorountine "Listen"
	// and waits for the sigterm
	// the boolean placed as a defined argument whether it is a test or not
	Start(bool) error

	TranslateCurrency(currency.CurrencySlice) []string

	// Builds a new message to send to the fetcher websocket
	NewMessage(string, []string, []string)

	// Handles the sigterm
	Interrupt()
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
	for _, fetcher := range fg.fetchers {
		fg.waitGroup.Add(1)

		go func(fetcher Fetcher) {
			defer fg.waitGroup.Done()
			fetcher.Start(false)
		}(fetcher)
	}

	fg.waitGroup.Wait()
}

// Sends message to each Fetcher's websocket
func (fg *FetcherGroup) SendMessage(aType string, productIds currency.CurrencySlice, channels []string) {
	for _, fetcher := range fg.fetchers {
		fetcher.NewMessage(aType, fetcher.TranslateCurrency(productIds), channels)
	}
}
