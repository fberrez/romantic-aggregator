package websocket

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"reflect"

	"github.com/gorilla/websocket"
)

type Proxy struct {
	Label  string
	WssUrl url.URL         `json:"wss_url"`
	Conn   *websocket.Conn `json:"conn"`

	// Channel which receive messages (sent by exchange) to the websocket
	MessageChannel chan []byte `json:"message_channel"`

	// Channel which send responses (sent by the websocket) to the exchange
	ResponseChannel chan []byte `json:"response_channel"`

	// Array which contains every message which have been sent
	Subscriptions [][]byte `json:"subscriptions"`
}

// Initializes the Proxy struct
func (p *Proxy) Initialize(uri url.URL, kafkaChannel chan interface{}) error {
	log.Printf("[%v Proxy] Initializing proxy\n", p.Label)
	c, _, err := websocket.DefaultDialer.Dial(uri.String(), nil)

	if err != nil {
		return err
	}

	p.WssUrl = uri
	p.Conn = c
	p.MessageChannel = make(chan []byte)
	p.ResponseChannel = make(chan []byte)
	p.Subscriptions = [][]byte{}

	return nil
}

// Starts the go routine which listen every messages sent by the Websocket
// and handles the following events :
//  - Receiving message in the MessageChannel
//  - Receiving a SIGINT
//  - Done chan closed by the listening go routine
func (p *Proxy) Start(testing bool) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})
	log.Printf("[%v Proxy] Connecting to %s\n", p.Label, p.WssUrl.String())

	// Listen and process every datas
	// received by the connection to the websocket.
	// If the connection is closed by the websocket,
	// the go routine is restarted with a new connection
	go p.Recoverer(-1, "ListenWebSocket", p.ListenWebsocket, done)

	// In case of testing, a SIGINT is send to the `interrupt` channel
	// to stop this process
	if testing {
		log.Printf("[%v Proxy] Closing test\n", p.Label)
		interrupt <- os.Interrupt
	}

	for {
		select {

		// If a new message arrives in the MessageChannel,
		// it is sent to the websocket
		case msg := <-p.MessageChannel:
			log.Printf("[%v Proxy] Sending message to Websocket: %v\n", p.Label, string(msg))
			if err := p.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("[%v Proxy] Error occured while sending message to Websocket: %v\n", p.Label, err)
				continue
			}

			// If a SIGINT is received, it closes the connection
		case <-interrupt:
			log.Printf("[%v Proxy] Closing Websocket\n", p.Label)
			err := p.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("[%v Proxy] Error occured while closing the Websocket: %v\n", p.Label, err)
				continue
			}
			return

			// 	// If the done channel is closed by the listening go routine
			// case <-done:
			// 	log.Printf("[%v Proxy] Closing Websocket because of an error\n", p.Label)
			// 	return
		}
	}
}

// Listens the websocket and processes datas it receives
// before to send it to the response channel
func (p *Proxy) ListenWebsocket(done chan struct{}) {
	log.Printf("[%v Proxy] Listenning to %s\n", p.Label, p.WssUrl.String())
	defer close(done)

	for {
		_, message, err := p.Conn.ReadMessage()
		if err != nil {
			log.Printf("[%v Proxy] Error occured while listening the websocket: %v\n\tmessage: %v\n", p.Label, err, message)
			return
		}

		p.ResponseChannel <- message
	}
}

// Handles if the listening go routine stopped
// Restarts it when it's occured
func (p *Proxy) Recoverer(maxPanics int, label string, f func(chan struct{}), done chan struct{}) {
	defer func() {
		if maxPanics == 0 {
			log.Printf("[%v Proxy] Too many panics on %v.\n", p.Label, label)
		} else {
			log.Printf("[%v Proxy] An error occured, Restarting %v.\n", p.Label, label)
			c, _, err := websocket.DefaultDialer.Dial(p.WssUrl.String(), nil)
			if err != nil {
				log.Printf("[%v Proxy] Cannot open a new connection.\n", p.Label)
				return
			}

			p.Conn = c
			done = make(chan struct{})
			for _, msg := range p.Subscriptions {
				p.MessageChannel <- msg
			}
			go p.Recoverer(maxPanics-1, "ListenWebsocket", f, done)
		}
	}()
	f(done)
}

// Returns false if at least one of these condition is verified:
// 	- The Proxy struct has not been initialized
// 	- The URL has not been initialized
// 	- The connection has not been initialized or is nil
func (p *Proxy) IsClean() error {
	if reflect.DeepEqual(p, &Proxy{}) {
		return fmt.Errorf("[%v Proxy] GDAX structure cannot be nil\n", p.Label)
	}

	if (p.WssUrl == url.URL{}) {
		return fmt.Errorf("[%v Proxy] %v structure doesn't have a good Websocket URL\n", p.Label, p.Label)
	}

	if p.Conn == nil || reflect.DeepEqual(p.Conn, &websocket.Conn{}) {
		return fmt.Errorf("[%v Proxy] %v structure doesn't have any connection etablished with the websocket\n", p.Label, p.Label)
	}

	return nil
}
