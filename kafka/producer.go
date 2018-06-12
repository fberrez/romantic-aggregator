package kafka

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"

	"github.com/Shopify/sarama"
)

const (
	TopicAggregator string = "romantic-aggregator"
)

// Contains a sarama AsyncProducer
// and a channel through which messages will be sent
// from the exchangers to the kafka stream
type AggregatorProducer struct {
	Producer sarama.AsyncProducer
	Channel  chan interface{}
}

// Initializes the Aggregator Producer
func Initialize(addr string) (*AggregatorProducer, error) {
	producer, err := sarama.NewAsyncProducer([]string{addr}, nil)
	if err != nil {
		return nil, err
	}

	aggrProd := &AggregatorProducer{
		Producer: producer,
		Channel:  make(chan interface{}),
	}

	return aggrProd, nil
}

// Starts the loop which will handle the messages stream and the sigterm
func (p *AggregatorProducer) Start() {
	defer p.Close()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

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
			log.Println("Failed to produce message", err)
			errors++
			// Handles sigterm
		case <-signals:
			break ProducerLoop
		}
	}

	log.Printf("Enqueued: %d; errors: %d\n", enqueued, errors)

}

// Sends messages to the kafka stream
func (p *AggregatorProducer) SendMessage(message interface{}) error {
	messageJSON, err := json.Marshal(message)

	if err != nil {
		return err
	}

	// Builds the message struct
	// which contains the topic name and the message
	producerMess := &sarama.ProducerMessage{
		Topic: TopicAggregator,
		Value: sarama.StringEncoder(string(messageJSON)),
	}

	p.Producer.Input() <- producerMess

	return nil
}

// Handles the sigterm
func (p *AggregatorProducer) Close() error {
	return p.Producer.Close()
}
