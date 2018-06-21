package websocket

import (
	"net/url"
	"reflect"

	"github.com/gorilla/websocket"
	"github.com/juju/errors"
	"github.com/sirupsen/logrus"
)

type Proxy struct {
	Label  string
	WssUrl url.URL         `json:"wss_url"`
	Conn   *websocket.Conn `json:"conn"`

	// Channel which receive messages (sent by exchange) to the websocket
	MessageChannel chan []byte `json:"message_channel"`

	// Channel which send responses (sent by the websocket) to the exchange
	ResponseChannel chan []byte `json:"response_channel"`

	// Receives the SIGINT
	InterruptChannel chan bool `json:"interrupt_channel"`

	// Current subscriptions in byte
	// (usefull when the websocket has been closed by the host)
	Subscriptions [][]byte `json:"subscriptions"`

	log *logrus.Entry
}

// Initializes the Proxy struct
func (p *Proxy) Initialize(uri url.URL) error {
	p.log = logrus.WithFields(logrus.Fields{"element": "proxy", "label": p.Label})

	p.log.Infof("Initializing proxy")

	c, _, err := websocket.DefaultDialer.Dial(uri.String(), nil)

	if err != nil {
		return err
	}

	p.WssUrl = uri
	p.Conn = c
	p.MessageChannel = make(chan []byte)
	p.ResponseChannel = make(chan []byte)
	p.InterruptChannel = make(chan bool)
	p.Subscriptions = [][]byte{}

	return nil
}

// Starts the go routine which listen every messages sent by the Websocket
// and handles the following events :
//  - Receiving message in the MessageChannel
//  - Receiving a SIGINT
//  - Done chan closed by the listening go routine
func (p *Proxy) Start() {
	p.log.WithFields(logrus.Fields{"url": p.WssUrl.String()}).Infof("Connecting...")

	// Listen and process every datas
	// received by the connection to the websocket.
	// If the connection is closed by the websocket,
	// the go routine is restarted with a new connection
	go p.ListenWebsocket()

	for {
		select {

		// If a new message arrives in the MessageChannel,
		// it is sent to the websocket
		case msg := <-p.MessageChannel:
			p.log.WithFields(logrus.Fields{"message": string(msg)}).Infof("Sending message to Websocket")
			if err := p.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				p.log.WithFields(logrus.Fields{"error": err}).Infof("Error occured while sending message to Websocket")
				continue
			}

			// If a SIGINT is received, it closes the connection
		case interrupt := <-p.InterruptChannel:
			if interrupt {
				p.log.Infof("Closing Websocket")
				err := p.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					p.log.WithFields(logrus.Fields{"error": err}).Errorf("Error occured while closing the Websocket")
					continue
				}
				return
			}
		}
	}
}

// Listens the websocket and processes datas it receives
// before to send it to the response channel
func (p *Proxy) ListenWebsocket() {
	p.log.Infof("Listening to %s", p.WssUrl.String())

ListeningLoop:
	for {
		_, message, err := p.Conn.ReadMessage()

		if err != nil {
			if !websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				p.log.WithFields(logrus.Fields{"message": err}).Infof("Websocket closed by client")
				break ListeningLoop
			}

			c, _, err := websocket.DefaultDialer.Dial(p.WssUrl.String(), nil)

			if err != nil {
				p.log.Error("Cannot open a new connection with %s. Closing %s.", p.WssUrl.String(), p.Label)
				break ListeningLoop
			}

			p.Conn = c
			for _, messageToSend := range p.Subscriptions {
				p.MessageChannel <- messageToSend
			}

			p.log.Warnf("Restarting websocket client")
			p.ListenWebsocket()

		}

		p.ResponseChannel <- message
	}
}

// Returns false if at least one of these condition is verified:
// 	- The Proxy struct has not been initialized
// 	- The URL has not been initialized
// 	- The connection has not been initialized or is nil
func (p *Proxy) IsClean() error {
	if reflect.DeepEqual(p, &Proxy{}) {
		return errors.NotAssignedf("%v proxy: structure cannot be nil", p.Label)
	}

	if (p.WssUrl == url.URL{}) {
		return errors.NotAssignedf("%v proxy: structure doesn't have a good Websocket URL", p.Label)
	}

	if p.Conn == nil || reflect.DeepEqual(p.Conn, &websocket.Conn{}) {
		return errors.NotAssignedf("%v proxy: structure doesn't have any connection etablished with the websocket", p.Label)
	}

	return nil
}

func (p *Proxy) Interrupt() {
	p.InterruptChannel <- true
}
