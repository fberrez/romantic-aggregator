package currency

import "fmt"

type CurrencyPair struct {
	FirstCurrency  string `json:"firstCurrency"`
	SecondCurrency string `json:"secondCurrency"`
}

type CurrencySlice []*CurrencyPair

var (
	BCHBTC *CurrencyPair = &CurrencyPair{"BCH", "BTC"}
	BCHUSD *CurrencyPair = &CurrencyPair{"BCH", "USD"}
	BTCEUR *CurrencyPair = &CurrencyPair{"BTC", "EUR"}
	BTCGBP *CurrencyPair = &CurrencyPair{"BTC", "GBP"}
	BTCUSD *CurrencyPair = &CurrencyPair{"BTC", "USD"}
	ETHBTC *CurrencyPair = &CurrencyPair{"ETH", "BTC"}
	ETHEUR *CurrencyPair = &CurrencyPair{"ETH", "EUR"}
	ETHUSD *CurrencyPair = &CurrencyPair{"ETH", "USD"}
	LTCBTC *CurrencyPair = &CurrencyPair{"LTC", "BTC"}
	LTCEUR *CurrencyPair = &CurrencyPair{"LTC", "EUR"}
)

func (c *CurrencyPair) ToGDAX() string {
	return fmt.Sprintf("%v-%v", c.FirstCurrency, c.SecondCurrency)
}

func (c CurrencySlice) ToGDAX() []string {
	result := []string{}

	for _, cp := range c {
		result = append(result, cp.ToGDAX())
	}

	return result
}

func (c *CurrencyPair) ToBitfinex() string {
	return fmt.Sprintf("t%v%v", c.FirstCurrency, c.SecondCurrency)
}

func (c CurrencySlice) ToBitfinex() []string {
	result := []string{}

	for _, cp := range c {
		result = append(result, cp.ToBitfinex())
	}

	return result
}
