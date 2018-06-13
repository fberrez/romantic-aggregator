package gdax

import (
	"time"

	"github.com/fberrez/romantic-aggregator/websocket"
)

type GDAX struct {
	Proxy         *websocket.Proxy `json:"proxy"`
	KafkaChannel  chan interface{} `json:"kafka_channel"`
	Subscriptions *Message         `json:"subscriptions"`
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

type HeartbeatResponse struct {
	Type        string    `json:"type"`
	Sequence    int       `json:"sequence"`
	LastTradeId int       `json:"last_trade_id"`
	ProductId   string    `json:"product_id"`
	Time        time.Time `json:"time"`
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

type SnapshotResponse struct {
	Type      string     `json:"type"`
	ProductId string     `json:"product_id"`
	Bids      [][]string `json:"bids"`
	Asks      [][]string `json:"asks"`
}

type L2UpdateResponse struct {
	Type      string     `json:"type"`
	ProductId string     `json:"product_id"`
	Time      time.Time  `json:"time"`
	Changes   [][]string `json:"changes"`
}
