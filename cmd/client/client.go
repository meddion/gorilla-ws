package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
	"github.com/meddion/gorilla-ws/pkg/types"
)

const URL = "ws://localhost:9999/ws"

func startClient(wg *sync.WaitGroup, symbols []types.Symbol) *ws.Conn {
	wg.Add(2)

	headers := http.Header{"origin": []string{"localhost"}}

	conn, _, err := ws.DefaultDialer.Dial(URL, headers)

	if err != nil {
		log.Fatalf("Error: %+v\n", err)
	}

	go func() {
		defer wg.Done()

		for {
			req := types.Request{
				Action:  "subscribe",
				Symbols: symbols,
			}

			err := conn.WriteJSON(&req)

			if err != nil {
				if err == ws.ErrCloseSent {
					log.Println("Closing on writing side...")
				} else {
					log.Printf("Error while writing to WS: %+v", err)
				}
				return
			}

			time.Sleep(time.Second)
		}
	}()

	go func() {
		defer wg.Done()

		for {
			var msg types.Message
			err := conn.ReadJSON(&msg)

			if err != nil {
				if ws.IsCloseError(err, ws.CloseNormalClosure) {
					log.Println("Closing on reading side...")
				} else {
					log.Printf("Error while reading from WS: %+v", err)
				}
				return
			}

			log.Printf("Read from the server: %+v\n", msg)
		}
	}()

	return conn
}

func main() {
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})
	var wg sync.WaitGroup

	conns := []*ws.Conn{
		startClient(&wg, []types.Symbol{types.LTCUSDT}),
		startClient(&wg, []types.Symbol{types.XBTUSDT}),
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-interrupt:
		for _, conn := range conns {
			err := conn.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("Error while closing: %+v", err)
			}
			conn.Close()
		}

		<-done

	case <-done:
	}
}
