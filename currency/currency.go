package currency

import (
	"fmt"

	"github.com/juju/errors"
)

type currencyPair struct {
	firstCurrency  string `json:"firstCurrency"`
	secondCurrency string `json:"secondCurrency"`
}

type CurrencySlice []*currencyPair

var (
	BCHBTC *currencyPair = &currencyPair{"BCH", "BTC"}
	BCHUSD *currencyPair = &currencyPair{"BCH", "USD"}
	BTCEUR *currencyPair = &currencyPair{"BTC", "EUR"}
	BTCGBP *currencyPair = &currencyPair{"BTC", "GBP"}
	BTCUSD *currencyPair = &currencyPair{"BTC", "USD"}
	ETHBTC *currencyPair = &currencyPair{"ETH", "BTC"}
	ETHEUR *currencyPair = &currencyPair{"ETH", "EUR"}
	ETHUSD *currencyPair = &currencyPair{"ETH", "USD"}
	LTCBTC *currencyPair = &currencyPair{"LTC", "BTC"}
	LTCEUR *currencyPair = &currencyPair{"LTC", "EUR"}

	AllCurrencies CurrencySlice = CurrencySlice{BCHBTC, BCHUSD, BTCEUR, BTCGBP, BTCUSD, ETHBTC, ETHEUR, ETHUSD, LTCBTC, LTCEUR}
)

// Tries to find a currency pair composed by `base` and `target`
func FindCurrencyPair(base string, target string) (*currencyPair, error) {
	if base == "" || target == "" {
		return nil, errors.NotValidf("To find a currency pair, the base and the target cannot be nil")
	}

	for _, currencyPair := range AllCurrencies {

		if currencyPair.firstCurrency == base && currencyPair.secondCurrency == target {
			return currencyPair, nil
		}

	}

	return nil, errors.NotFoundf("Currency Pair %v-%v not found.", base, target)
}

// Formats a currency pair to GDAX
func (c *currencyPair) ToGDAX() (string, error) {
	if c.firstCurrency == "" || c.secondCurrency == "" {
		return "", errors.NotValidf("A currency pair must be correctly initiliazed: base and target cannot be nil")
	}

	return fmt.Sprintf("%v-%v", c.firstCurrency, c.secondCurrency), nil
}

// Formats a currency slice to GDAX
func (c CurrencySlice) ToGDAX() ([]string, error) {
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

// Formats a currency pair to Bitfinex
func (c *currencyPair) ToBitfinex() (string, error) {
	if c.firstCurrency == "" || c.secondCurrency == "" {
		return "", errors.NotValidf("A currency pair must be correctly initiliazed: base and target cannot be nil")
	}

	return fmt.Sprintf("%v%v", c.firstCurrency, c.secondCurrency), nil
}

// Formats a currency slice to Bitfinex
func (c CurrencySlice) ToBitfinex() ([]string, error) {
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
