package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/meddion/gorilla-ws/pkg/broker"
	"github.com/meddion/gorilla-ws/pkg/publisher"
	"github.com/meddion/gorilla-ws/pkg/subscriber"
	"github.com/meddion/gorilla-ws/pkg/types"
	"golang.org/x/sync/errgroup"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("origin") == "localhost"
	},
}

func createBrokerSubHandler(brk types.Broker) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "on upgrading to a WS connection"})
			return
		}

		sub := subscriber.NewBitmexSubscriber(conn, brk)

		if err := sub.Start(); err != nil {
			c.Error(err)
		}
	}
}

func createPublisher(authCfg types.AuthConfig) types.Publisher {
	topics := []types.Topic{types.Instrument}
	url := authCfg.URL + authCfg.Endpoint

	return publisher.NewBitmexPublisher(url, &authCfg, topics)
}

func createHTTPServer(brk types.Broker) *http.Server {
	r := gin.Default()

	r.Use(gin.Logger())
	r.GET("/ws", createBrokerSubHandler(brk))

	return &http.Server{
		Addr:    ":9999",
		Handler: r,
	}
}

// TODO: add a graceful shutdown on os.Signal
func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("On parsing the .env file: %+v\n", err)
	}
	authCfg := types.AuthConfig{
		URL:       os.Getenv("URL"),
		ApiKey:    os.Getenv("API_KEY"),
		ApiSecret: os.Getenv("API_SECRET"),
		Endpoint:  os.Getenv("ENDPOINT"),
		Verb:      os.Getenv("VERB"),
	}

	pub := createPublisher(authCfg)
	bitmexBroker := broker.NewBitmexBroker()
	srv := createHTTPServer(bitmexBroker)

	g := new(errgroup.Group)

	g.Go(func() error {
		if err := pub.StartPublishing(bitmexBroker); err != nil {
			return err
		}
		log.Printf("Publisher has stopped.")

		return nil
	})

	g.Go(func() error {
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			return err
		}
		log.Printf("HTTP-Server has stopped.")

		return nil
	})

	if err := g.Wait(); err != nil {
		log.Printf("Errror from the errorgroup: %+v\n", err)
	}
}
