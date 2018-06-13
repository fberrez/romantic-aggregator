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

	log.Printf("[%v Proxy] Connecting to %s\n", p.Label, p.WssUrl.String())

	// Listen and process every datas
	// received by the connection to the websocket.
	// If the connection is closed by the websocket,
	// the go routine is restarted with a new connection
	go p.Recoverer(p.ListenWebsocket)

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
		}
	}
}

// Listens the websocket and processes datas it receives
// before to send it to the response channel
func (p *Proxy) ListenWebsocket() {
	log.Printf("[%v Proxy] Listenning to %s\n", p.Label, p.WssUrl.String())

	for {
		_, message, err := p.Conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				panic(fmt.Sprintf("[%v Proxy] Error occured while listening the websocket: %v\n\tmessage: %v\n", p.Label, err, message))
			} else {
				log.Printf("[%v Proxy] Websocket closed by client: %s", p.Label, err)
			}
		}

		p.ResponseChannel <- message
	}
}

func (p *Proxy) Recoverer(f func()) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[%s Proxy] An error occured in the recoverer:\n\t- err: %v\n", p.Label, err)
			c, _, err := websocket.DefaultDialer.Dial(p.WssUrl.String(), nil)

			if err != nil {
				log.Printf("[%s Proxy] Cannot open a new connection with %s. Closing %s.", p.Label, p.WssUrl.String(), p.Label)
			}

			p.Conn = c
			for _, messageToSend := range p.Subscriptions {
				p.MessageChannel <- messageToSend
			}
			go p.Recoverer(f)

		}
	}()
	f()
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
