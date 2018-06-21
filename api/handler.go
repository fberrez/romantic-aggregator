package api

import (
	"fmt"

	"github.com/fberrez/romantic-aggregator/currency"
	"github.com/fberrez/romantic-aggregator/exchange"
	"github.com/gin-gonic/gin"
)

type SubscribeIn struct {
	Base   string `path:"base" validate="required"`
	Target string `path:"target" validate="required"`
	Action string `path:"action" enum:"subscribe,unsubscribe" validate="required"`
}

var (
	subscribe   string = "subscribe"
	unsubscribe string = "unsubscribe"
)

// Handles requests sent to /ticker/{base}/{target}/{action}
func (a *Api) subscribeHandler(c *gin.Context, in *SubscribeIn) error {
	currencyPair, err := currency.FindCurrencyPair(in.Base, in.Target)

	if err != nil {
		return err
	}

	errorsArray := []error{}

	switch in.Action {
	case subscribe:
		errorsArray = a.FetcherGroup.SendMessage(exchange.Subscribe, currency.CurrencySlice{currencyPair}, []string{"ticker"})
	case unsubscribe:
		errorsArray = a.FetcherGroup.SendMessage(exchange.Unsubscribe, currency.CurrencySlice{currencyPair}, []string{"ticker"})
	}

	if len(errorsArray) > 0 {
		return errorsArray[0]
	}

	message := fmt.Sprintf("subscription to %s%s done", in.Base, in.Target)
	c.JSON(200, gin.H{"message": message})

	return nil
}
