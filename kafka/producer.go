package kafka

import (
	"encoding/json"

	"github.com/Shopify/sarama"
	"github.com/sirupsen/logrus"
)

const (
	TopicAggregator string = "romantic-aggregator"
)

// Contains a sarama AsyncProducer
// and a channel through which messages will be sent
// from the exchangers to the kafka stream
type AggregatorProducer struct {
	Producer         sarama.AsyncProducer
	Channel          chan interface{}
	InterruptChannel chan bool
}

var (
	log *logrus.Entry = logrus.WithFields(logrus.Fields{"element": "kafka"})
)

// Initializes the Aggregator Producer
func Initialize(addr string) (*AggregatorProducer, error) {
	producer, err := sarama.NewAsyncProducer([]string{addr}, nil)
	if err != nil {
		return nil, err
	}

	aggrProd := &AggregatorProducer{
		Producer:         producer,
		Channel:          make(chan interface{}),
		InterruptChannel: make(chan bool),
	}

	return aggrProd, nil
}

// Starts the loop which will handle the messages stream and the sigterm
func (p *AggregatorProducer) Start() {
	defer p.Stop()

	var enqueued, errors int

ProducerLoop:
	for {
		select {
		// Handles messages received from the channel
		// between exchangers and kafka producer
		case message := <-p.Channel:
			p.SendMessage(message)
			// Handles errors
		case err := <-p.Producer.Errors():
			log.WithFields(logrus.Fields{"error": err}).Errorf("Failed to produce message")
			errors++
			// Handles sigterm
		case signal := <-p.InterruptChannel:
			if signal {
				break ProducerLoop
			}
		}
	}

	log.WithFields(logrus.Fields{"enqueued": enqueued, "errors": errors}).Infof("Closing Kafka producer")

}

// Sends messages to the kafka stream
func (p *AggregatorProducer) SendMessage(message interface{}) error {
	marshalledMessage, err := json.Marshal(message)

	if err != nil {
		return err
	}

	// Builds the message struct
	// which contains the topic name and the message
	producerMess := &sarama.ProducerMessage{
		Topic: TopicAggregator,
		Value: sarama.StringEncoder(string(marshalledMessage)),
	}

	p.Producer.Input() <- producerMess

	return nil
}

// Handles the sigterm
func (p *AggregatorProducer) Stop() error {
	p.InterruptChannel <- true
	return p.Producer.Close()
}
