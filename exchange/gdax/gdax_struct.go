package gdax

import (
	"github.com/fberrez/romantic-aggregator/aggregator"
	"github.com/fberrez/romantic-aggregator/websocket"
)

type GDAX struct {
	Proxy             *websocket.Proxy             `json:"proxy"`
	AggregatorChannel chan aggregator.SimpleTicker `json:"aggregator_channel"`
	Subscriptions     *Message                     `json:"subscriptions"`
	InterruptChannel  chan bool                    `json:"interrupt_channel"`
}

type Message struct {
	Type       string   `json:"type"`
	ProductIds []string `json:"product_ids"`
	Channels   []string `json:"channels"`
}

type SubscriptionResponse struct {
	Type     string                `json:"type"`
	Channels []ChannelSubscription `json:"channels"`
}

type ChannelSubscription struct {
	Name       string   `json:"name"`
	ProductIds []string `json:"product_ids"`
}

type Response struct {
	Type string `json:"type"`
}

type TickerResponse struct {
	Type      string `json:"type"`
	Sequence  int    `json:"sequence"`
	ProductId string `json:"product_id"`
	Price     string `json:"price"`
	Open24h   string `json:"open_24h"`
	Volume24h string `json:"volume_24h"`
	Low24h    string `json:"low_24h"`
	High24h   string `json:"high_24h"`
	Volume30d string `json:"volume_30d"`
	BestBid   string `json:"best_bid"`
	BestAsk   string `json:"best_ask"`
}

type TickerApiResponse struct {
	TradeId int    `json:"trade_id"`
	Price   string `json:"price"`
	Size    string `json:"size"`
	Bid     string `json:"bid"`
	Ask     string `json:"ask"`
	Volume  string `json:"volume"`
	Time    string `json:"time"`
}
