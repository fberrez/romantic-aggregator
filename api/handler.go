package api

import (
	"fmt"

	"github.com/fberrez/romantic-aggregator/aggregator"
	"github.com/fberrez/romantic-aggregator/currency"
	"github.com/fberrez/romantic-aggregator/exchange"
	"github.com/gin-gonic/gin"
)

type SubscribeIn struct {
	Base   string `path:"base" validate="required"`
	Target string `path:"target" validate="required"`
	Action string `path:"action" enum:"subscribe,unsubscribe" validate="required"`
}

type TimerIn struct {
	New string `path:"new" enum:"1m,3m,5m,15m,30m,45m,1H,2H,3H,4H,1D,1W,1M" validate="required"`
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

func (a *Api) timerHandler(c *gin.Context, in *TimerIn) error {
	switch in.New {
	case "1m":
		a.aggregator.SetInterval(aggregator.OneMinute)
	case "3m":
		a.aggregator.SetInterval(aggregator.ThreeMinutes)
	case "5m":
		a.aggregator.SetInterval(aggregator.FiveMinutes)
	case "15m":
		a.aggregator.SetInterval(aggregator.FifTeenMinutes)
	case "30m":
		a.aggregator.SetInterval(aggregator.ThirtyMinutes)
	case "45m":
		a.aggregator.SetInterval(aggregator.FortyFiveMinutes)
	case "1H":
		a.aggregator.SetInterval(aggregator.OneHour)
	case "2H":
		a.aggregator.SetInterval(aggregator.TwoHours)
	case "3H":
		a.aggregator.SetInterval(aggregator.ThreeHours)
	case "4H":
		a.aggregator.SetInterval(aggregator.FourHours)
	case "1D":
		a.aggregator.SetInterval(aggregator.OneDay)
	case "1W":
		a.aggregator.SetInterval(aggregator.OneWeek)
	case "1M":
		a.aggregator.SetInterval(aggregator.OneMonth)
	}

	message := fmt.Sprintf("Time set on %s", in.New)

	c.JSON(200, gin.H{"message": message})
	return nil
}
