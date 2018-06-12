package bitfinex

import (
	"net/url"

	"github.com/gorilla/websocket"
)

type Bitfinex struct {
	WssUrl       url.URL          `json:"wss_url"`
	Conn         *websocket.Conn  `json:"conn"`
	MessageQueue []Message        `json:"message_queue"`
	KafkaChannel chan interface{} `json:"kafka_channel"`
}

type Message struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	Symbol  string `json:"symbol"`
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
