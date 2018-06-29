package main

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/fberrez/romantic-aggregator/api"
	"github.com/sirupsen/logrus"
)

var (
	log *logrus.Entry = logrus.WithFields(logrus.Fields{"element": "romantic-aggregator"})
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
}

func main() {
	start := time.Now()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	port := os.Getenv("API_PORT")

	if port == "" {
		port = "4242"
	}

	a := api.Initiliaze()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: a.Fizz,
	}

	waitGroupApi := sync.WaitGroup{}

	a.Start(waitGroupApi)

	go func() {
		for {
			select {
			case <-interrupt:
				a.Stop()
				waitGroupApi.Wait()
				log.Info("Stopping server")
				err := srv.Close()

				if err != nil {
					log.Error("Error server: %v", err)
					os.Exit(1)
				}

				os.Exit(0)
			}
		}
	}()

	log.WithField("port", ":"+port).Info("Launching server")
	elapsed := time.Since(start)
	log.Infof("Server started in %s", elapsed)

	err := srv.ListenAndServe()

	if err != nil && err.Error() != "http: Server closed" {
		log.WithField("error", err).Errorf("An error occured while launching the server")
	}
}
