package currency

import (
	"testing"

	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
)

func TestFindCurrencyPair(t *testing.T) {
	tables := []struct {
		base   string
		target string
		result *currencyPair
		err    string
	}{
		{"BTC", "GBP", &currencyPair{"BTC", "GBP"}, ""},
		{"BTC", "", nil, "notValid"},
		{"", "EUR", nil, "notValid"},
		{"", "", nil, "notValid"},
		{"LTC", "GBP", nil, "notFound"},
	}

	for _, table := range tables {
		result, err := FindCurrencyPair(table.base, table.target)

		assert.Equal(t, table.result, result)

		switch table.err {
		case "notFound":
			assert.True(t, errors.IsNotFound(err))
		case "notValid":
			assert.True(t, errors.IsNotValid(err))
		case "":
			assert.Nil(t, err)
		}
	}
}

func TestToGDAX(t *testing.T) {
	tables := []struct {
		slice  CurrencySlice
		result []string
		err    string
	}{
		{CurrencySlice{BCHBTC, BTCEUR, BTCGBP}, []string{"BCH-BTC", "BTC-EUR", "BTC-GBP"}, ""},
		{CurrencySlice{BCHBTC, BTCEUR, &currencyPair{"BTC", "LTC"}}, []string{"BCH-BTC", "BTC-EUR", "BTC-LTC"}, ""},
		{CurrencySlice{&currencyPair{"BTC", "LTC"}}, []string{"BTC-LTC"}, ""},
		{CurrencySlice{&currencyPair{"", ""}}, []string{}, "notValid"},
		{CurrencySlice{BTCEUR, &currencyPair{"", ""}}, []string{"BTC-EUR"}, "notValid"},
	}

	for _, table := range tables {
		result, err := table.slice.ToGDAX()

		assert.Equal(t, table.result, result)

		switch table.err {
		case "notValid":
			assert.True(t, errors.IsNotValid(err))
		}
	}
}

func TestToBitfinex(t *testing.T) {
	tables := []struct {
		slice  CurrencySlice
		result []string
		err    string
	}{
		{CurrencySlice{BCHBTC, BTCEUR, BTCGBP}, []string{"BCHBTC", "BTCEUR", "BTCGBP"}, ""},
		{CurrencySlice{BCHBTC, BTCEUR, &currencyPair{"BTC", "LTC"}}, []string{"BCHBTC", "BTCEUR", "BTCLTC"}, ""},
		{CurrencySlice{&currencyPair{"BTC", "LTC"}}, []string{"BTCLTC"}, ""},
		{CurrencySlice{&currencyPair{"", ""}}, []string{}, "notValid"},
		{CurrencySlice{BTCEUR, &currencyPair{"", ""}}, []string{"BTCEUR"}, "notValid"},
	}

	for _, table := range tables {
		result, err := table.slice.ToBitfinex()

		assert.Equal(t, table.result, result)

		switch table.err {
		case "notValid":
			assert.True(t, errors.IsNotValid(err))
		}
	}
}

func TestToString(t *testing.T) {
	tables := []struct {
		slice  CurrencySlice
		result string
	}{
		{CurrencySlice{BCHBTC, BTCEUR, BTCGBP}, "{BCH BTC} - {BTC EUR} - {BTC GBP}"},
		{CurrencySlice{BCHBTC, BTCEUR, &currencyPair{"BTC", "LTC"}}, "{BCH BTC} - {BTC EUR} - {BTC LTC}"},
		{CurrencySlice{&currencyPair{"BTC", "LTC"}}, "{BTC LTC}"},
		{CurrencySlice{&currencyPair{"", ""}}, "{ }"},
		{CurrencySlice{BTCEUR, &currencyPair{"", ""}}, "{BTC EUR} - { }"},
	}

	for _, table := range tables {
		result := table.slice.ToString()

		assert.Equal(t, table.result, result)
	}

}
