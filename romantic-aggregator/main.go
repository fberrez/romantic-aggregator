package main

import (
	"net/http"
	"os"
	"os/signal"
	"sync"

	"github.com/fberrez/romantic-aggregator/api"
	"github.com/sirupsen/logrus"
)

var (
	log *logrus.Entry = logrus.WithFields(logrus.Fields{"element": "romantic-aggregator"})
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	a := api.Initiliaze()

	srv := &http.Server{
		Addr:    ":4242",
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

	srv.ListenAndServe()

}
