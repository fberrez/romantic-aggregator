package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/fberrez/romantic-aggregator/currency"
	"github.com/fberrez/romantic-aggregator/exchange"
	"github.com/fberrez/romantic-aggregator/kafka"
)

func main() {
	waitGroup := sync.WaitGroup{}

	kafkaAddr := os.Getenv("KAFKA_ADDRESS")

	if kafkaAddr == "" {
		log.Fatal("Please, export KAFKA_ADDRESS. Try `export KAFKA_ADDRESS=XXX.XXX.XXX.XXX:9092` OR `-e KAFKA_ADDRESS=XXX.XXX.XXX.XXX:9092` if you're running it with Docker")
	}

	producer, err := kafka.Initialize(kafkaAddr)

	if err != nil {
		panic(err)
	}

	fg := exchange.Initialize(producer.Channel)

	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()
		producer.Start()
	}()

	go func() {
		defer waitGroup.Done()
		fg.Start()
	}()

	fmt.Printf("\n\n------------------------------ SUBSCRIBE ETHEUR ------------------------------------\n\n")
	fg.SendMessage(exchange.Subscribe, currency.CurrencySlice{currency.ETHEUR}, []string{"ticker"})

	time.Sleep(2 * time.Second)
	fmt.Printf("\n\n------------------------------ SUBSCRIBE BTCEUR ------------------------------------\n\n")
	fg.SendMessage(exchange.Subscribe, currency.CurrencySlice{currency.BTCEUR}, []string{"ticker"})

	time.Sleep(4 * time.Second)
	fmt.Printf("\n\n------------------------------ UNSUBSCRIBE ETHEUR ------------------------------------\n\n")
	fg.SendMessage(exchange.Unsubscribe, currency.CurrencySlice{currency.ETHEUR}, []string{"ticker"})
	waitGroup.Wait()
}
