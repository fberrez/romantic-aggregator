package bitfinex

import (
	"testing"

	"github.com/fberrez/romantic-aggregator/aggregator"
	"github.com/stretchr/testify/assert"
)

// tables := []struct{
//
//   }{
//   {},
// }
//
// for _, table := range tables {
//
// }

var (
	emptyBitfinex       *Bitfinex = &Bitfinex{}
	initializedBitfinex *Bitfinex = generateNewBitfinex()

	normalKafkaChan chan aggregator.SimpleTicker = make(chan aggregator.SimpleTicker)
)

func TestInitialize(t *testing.T) {
	tables := []struct {
		bitfinex       *Bitfinex
		aggregatorChan chan aggregator.SimpleTicker
		err            string
	}{
		{emptyBitfinex, normalKafkaChan, "nil"},
	}

	for _, table := range tables {
		err := table.bitfinex.Initialize(table.aggregatorChan)

		switch table.err {
		case "nil":
			assert.Nil(t, err)
		}
	}
}

func generateNewBitfinex() *Bitfinex {
	b := &Bitfinex{}
	b.Initialize(make(chan aggregator.SimpleTicker))

	return b
}
