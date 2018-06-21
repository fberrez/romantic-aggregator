package bitfinex

import (
	"github.com/fberrez/romantic-aggregator/aggregator"
	"github.com/fberrez/romantic-aggregator/websocket"
)

type Bitfinex struct {
	Proxy             *websocket.Proxy             `json:"proxy"`
	AggregatorChannel chan aggregator.SimpleTicker `json:"aggregator_channel"`
	Subscriptions     []SubscribeResponse          `json:"subscriptions"`
	InterruptChannel  chan bool                    `json:"interrupt_channel"`
}

type Message struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	Symbol  string `json:"symbol"`
}

type UnsubscribeMessage struct {
	Event  string `json:"event"`
	ChanId int    `json:"chanId"`
}

type TickerResponse struct {
	ChannelId       int     `json:"channel_id"`
	Bid             float64 `json:"bid"`
	BidSize         float64 `json:"bid_size"`
	Ask             float64 `json:"ask"`
	AskSize         float64 `json:"ask_size"`
	DailyChange     float64 `json:"daily_change"`
	DailyChangePrec float64 `json:"daily_change_prec"`
	LastPrice       float64 `json:"last_price"`
	Volume          float64 `json:"volume"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
}

type SubscribeResponse struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	ChanId  int    `json:"chanId"`
	Pair    string `json:"pair"`
}

type UnsubscribeResponse struct {
	Event  string `json:"event"`
	Status string `json:"status"`
	ChanId int    `json:"ChanId"`
}
