package gdax

import (
	"encoding/json"
	"testing"

	"github.com/gorilla/websocket"
)

func TestInitialize(t *testing.T) {
	c, _, _ := websocket.DefaultDialer.Dial(uri.String(), nil)
	exceptedGdax := &GDAX{WssUrl: uri, MessageQueue: []Message{}, Conn: c, KafkaChannel: make(chan interface{})}

	tables := []struct {
		gdax     *GDAX
		channel  chan interface{}
		hasError bool
	}{
		{&GDAX{WssUrl: uri, MessageQueue: []Message{}, Conn: c, KafkaChannel: make(chan interface{})}, make(chan interface{}), false},
	}

	for index, table := range tables {
		err := table.gdax.Initialize(table.channel)

		if (err != nil) != table.hasError {
			t.Errorf("[TEST (*GDAX).New #%v]A non-excepted error occured. Err: %v\n", index, err)
		}

		if table.gdax.WssUrl != table.gdax.WssUrl || len(table.gdax.MessageQueue) != len(table.gdax.MessageQueue) && table.gdax.Conn != table.gdax.Conn {
			gdaxJson, _ := json.Marshal(table.gdax)
			exceptedGdaxJson, _ := json.Marshal(exceptedGdax)
			t.Errorf("[TEST (*GDAX).New #%v] Bad GDAX struct returned,\n\t got: %v,\n\t want: %v\n", index, string(gdaxJson), string(exceptedGdaxJson))
		}

		table.gdax.Interrupt()
	}

	c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

func TestStart(t *testing.T) {
	c1, _, _ := websocket.DefaultDialer.Dial(uri.String(), nil)
	c2, _, _ := websocket.DefaultDialer.Dial(uri.String(), nil)
	c3, _, _ := websocket.DefaultDialer.Dial(uri.String(), nil)
	c4, _, _ := websocket.DefaultDialer.Dial(uri.String(), nil)

	waitingMessage := Message{
		Type:       "subscribe",
		ProductIds: []string{"ETH-USD"},
		Channels:   []string{"heartbeat"},
	}

	tables := []struct {
		gdax     *GDAX
		hasError bool
	}{
		{&GDAX{WssUrl: uri, MessageQueue: []Message{}, Conn: c1, KafkaChannel: make(chan interface{})}, false},
		{&GDAX{WssUrl: uri, MessageQueue: []Message{waitingMessage}, Conn: c1, KafkaChannel: make(chan interface{})}, false},
		{&GDAX{}, true},
		{&GDAX{MessageQueue: []Message{}, Conn: c2, KafkaChannel: make(chan interface{})}, true},
		{&GDAX{WssUrl: uri, Conn: c3, KafkaChannel: make(chan interface{})}, true},
		{&GDAX{WssUrl: uri, MessageQueue: []Message{}, KafkaChannel: make(chan interface{})}, true},
		{&GDAX{WssUrl: uri, MessageQueue: []Message{}, Conn: c4}, true},
	}

	for index, table := range tables {
		err := table.gdax.Start(true)

		if (err != nil) != table.hasError {
			t.Errorf("[TEST (*GDAX).Start #%v] A non-excepted error occured. Err: %v ", index, err)
		}
	}
}
