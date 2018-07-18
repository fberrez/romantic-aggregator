package api

import (
	"os"
	"sync"
	"time"

	"github.com/fberrez/romantic-aggregator/aggregator"
	"github.com/fberrez/romantic-aggregator/exchange"
	"github.com/fberrez/romantic-aggregator/kafka"
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/sirupsen/logrus"
	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// Contains each part of the aggregator
type Api struct {
	// It is the server which handling http routes of the API
	Fizz *fizz.Fizz
	// Contains exchanges informations and variables
	FetcherGroup *exchange.FetcherGroup
	// Receives the SIGINT
	InterruptChannel []chan bool

	producer   *kafka.AggregatorProducer
	aggregator *aggregator.Aggregator
}

var (
	log = logrus.WithFields(logrus.Fields{"element": "api"})
)

const (
	maxNumberOfTest = 5
	initialDelay    = 2
)

// Initializes the api with a new struct and initialized routes
func Initiliaze() *Api {
	engine := gin.New()

	f := fizz.NewFromEngine(engine)

	api := &Api{
		Fizz:             f,
		InterruptChannel: make([]chan bool, 1),
	}

	infos := &openapi.Info{
		Title:       "Romantic Aggregator",
		Description: "The purpose of the aggregator is to retrieve datas (ticker) about the currencies to which the aggregrator has subscribed. Once recovered, the datas are sent to a Kafka stream.",
		Version:     "0.0.1",
	}

	f.GET("/openapi.json", nil, f.OpenAPI(infos, "json"))
	f.GET("/ticker/:base/:target/:action", nil, tonic.Handler(api.subscribeHandler, 200))
	f.GET("/timer/:new", nil, tonic.Handler(api.timerHandler, 200))

	return api
}

// Initializes the aggregator
func InitializeAggregator(kafkaChan chan interface{}) *aggregator.Aggregator {
	aggregator := aggregator.Initialize(kafkaChan)

	return aggregator
}

// Initializes the kafka producer
func InitializeProducer() *kafka.AggregatorProducer {
	kafkaAddr := os.Getenv("KAFKA_ADDRESS")

	if kafkaAddr == "" {
		log.Fatal("Please, export KAFKA_ADDRESS. Try `export KAFKA_ADDRESS=XXX.XXX.XXX.XXX:9092` OR `-e KAFKA_ADDRESS=XXX.XXX.XXX.XXX:9092` if you're running it with Docker")
	}

	producer, err := kafka.Initialize(kafkaAddr)

	delay := initialDelay
	currentNumberOfTestRemaining := maxNumberOfTest

	// while an error occured when initializing the kafka producer,
	// tries to reinitialize a new one until the number of test reach 0
	for err != nil {
		if currentNumberOfTestRemaining == 0 {
			panic(err)
		}

		log.WithField("test-remaining", currentNumberOfTestRemaining).Warningf("Initializing the kafka producer failed. Retrying in %d seconds", delay)
		time.Sleep(time.Duration(delay) * time.Second)

		producer, err = kafka.Initialize(kafkaAddr)
		currentNumberOfTestRemaining--
		delay *= 2
	}

	return producer
}

// Starts api and its services (kafka producer, exchanges, aggregator)
func (a *Api) Start(waitGroup sync.WaitGroup) {

	a.producer = InitializeProducer()
	a.aggregator = InitializeAggregator(a.producer.Channel)
	a.FetcherGroup = exchange.Initialize(a.aggregator.AggregatorChannel)

	waitGroup.Add(3)

	go func() {
		defer waitGroup.Done()
		a.producer.Start()
	}()

	go func() {
		defer waitGroup.Done()
		a.FetcherGroup.Start()
	}()

	go func() {
		defer waitGroup.Done()
		a.aggregator.Start()
	}()

}

// Stops api and its services (kafka producer, exchanges, aggregator)
func (a *Api) Stop() {
	a.producer.Stop()
	a.FetcherGroup.Stop()
	a.aggregator.Stop()
}
