package exchange

import (
	"sync"

	"github.com/fberrez/romantic-aggregator/aggregator"
	"github.com/fberrez/romantic-aggregator/currency"
	"github.com/fberrez/romantic-aggregator/exchange/bitfinex"
	"github.com/fberrez/romantic-aggregator/exchange/gdax"
	"github.com/sirupsen/logrus"
)

type Fetcher interface {
	// Initializes the Fetcher and his websocket dialer
	Initialize(chan aggregator.SimpleTicker) error

	// Launches the gorountine "Listen"
	// and waits for the sigterm
	// the boolean placed as a defined argument whether it is a test or not
	Start() error

	// Translates a CurrencySlice (which contains CurrencyPair) to an array of strings.
	// Adapts each string the specificities of each platform
	// (ex: GDAX = "BTC-USD", Bitfinex = "tBTCUSD", ...)
	TranslateCurrency(currency.CurrencySlice) ([]string, error)

	// Builds a new message to send to the fetcher websocket
	NewMessage(bool, []string, []string) error

	// Interrupt exchanges and Proxy
	Interrupt()
}

// FetcherGroup contains an array of Fetcher
// and a WaitGroup (which waits for a collection of goroutines to finish)
type FetcherGroup struct {
	fetchers          []Fetcher
	waitGroup         sync.WaitGroup
	exchangeChannel   chan aggregator.SimpleTicker
	aggregatorChannel chan aggregator.SimpleTicker
}

var (
	Drivers = map[string]Fetcher{
		"GDAX":     &gdax.GDAX{},
		"Bitfinex": &bitfinex.Bitfinex{},
	}
	log *logrus.Entry = logrus.WithFields(logrus.Fields{"element": "exchange"})
)

const (
	Subscribe   bool = true
	Unsubscribe bool = false
)

// Initializes a FetcherGroup and
// each Fetcher which are in the FetcherGroup's fetchers
func Initialize(aggregatorChan chan aggregator.SimpleTicker) *FetcherGroup {
	fg := &FetcherGroup{
		fetchers:          make([]Fetcher, 0),
		waitGroup:         sync.WaitGroup{},
		aggregatorChannel: aggregatorChan,
	}

	for driverName, driver := range Drivers {
		err := driver.Initialize(fg.aggregatorChannel)

		if err != nil {
			log.WithFields(logrus.Fields{"error": err}).Errorf("Initializing %s", driverName)
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
			err := fetcher.Start()

			if err != nil {
				log.WithFields(logrus.Fields{"error": err}).Errorf("Trying to start #%v", index)
			}
		}(fetcher)
	}

	fg.waitGroup.Wait()
}

// Sends message to each Fetcher's websocket
func (fg *FetcherGroup) SendMessage(isSubscribe bool, productIds currency.CurrencySlice, channels []string) []error {
	errors := []error{}

	log.WithFields(logrus.Fields{
		"subscribe":   isSubscribe,
		"product_ids": productIds.ToString(),
		"channels":    channels,
	}).Info("Update subscriptions")

	for index, fetcher := range fg.fetchers {
		formattedCurrencie, err := fetcher.TranslateCurrency(productIds)

		if err != nil {
			log.WithFields(logrus.Fields{"error": err}).Errorf("Trying to send a message to #%d", index)
			errors = append(errors, err)
		}

		err = fetcher.NewMessage(isSubscribe, formattedCurrencie, channels)

		if err != nil {
			log.WithFields(logrus.Fields{"error": err}).Errorf("Trying to send a message to #%d:", index)
			errors = append(errors, err)
		}
	}

	return errors
}

func (fg *FetcherGroup) Stop() {
	log.Info("Closing exchanges")
	for _, fetcher := range fg.fetchers {
		fetcher.Interrupt()
	}
}
