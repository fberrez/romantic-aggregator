package api

import (
	"os"
	"sync"

	"github.com/fberrez/romantic-aggregator/aggregator"
	"github.com/fberrez/romantic-aggregator/exchange"
	"github.com/fberrez/romantic-aggregator/kafka"
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/sirupsen/logrus"
	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

type Api struct {
	Fizz             *fizz.Fizz
	producer         *kafka.AggregatorProducer
	aggregator       *aggregator.Aggregator
	FetcherGroup     *exchange.FetcherGroup
	InterruptChannel []chan bool
}

var (
	log *logrus.Entry = logrus.WithFields(logrus.Fields{"element": "api"})
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

	if err != nil {
		panic(err)
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
