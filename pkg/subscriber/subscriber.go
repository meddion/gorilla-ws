package subscriber

import (
	"sort"
	"sync"

	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
	"github.com/meddion/gorilla-ws/pkg/types"
)

const SYMBOLS_CAPACITY = 10

type BitmexSubscriber struct {
	sync.Once
	sync.RWMutex
	conn     *ws.Conn
	broker   types.Broker
	recvChan chan types.Message
	done     chan struct{}
	symbols  []types.Symbol

	id string
	// TODO: change it to a slice
	err error
}

func NewBitmexSubscriber(conn *ws.Conn, broker types.Broker) *BitmexSubscriber {
	return &BitmexSubscriber{
		id:       uuid.NewString(),
		conn:     conn,
		broker:   broker,
		recvChan: make(chan types.Message, 10),
		done:     make(chan struct{}),
		symbols:  make([]types.Symbol, 0, SYMBOLS_CAPACITY),
	}
}

func (b *BitmexSubscriber) GetID() string {
	return b.id
}

func (b *BitmexSubscriber) Receive(msg types.Message) {
	select {
	case b.recvChan <- msg:
	case <-b.done:
	}
}

func (b *BitmexSubscriber) getError() error {
	b.RLock()
	defer b.RUnlock()

	return b.err
}

func (b *BitmexSubscriber) setError(err error) {
	b.Lock()
	defer b.Unlock()
	b.err = err
}

// TODO:  handle errors prroperly
// ATTENTION: very ugly code ahead
func (b *BitmexSubscriber) unsubscribe(symbols []types.Symbol) error {
	b.Lock()
	defer b.Unlock()

	for _, symb := range symbols {
		for i, storedSymb := range b.symbols {
			if storedSymb == symb {
				b.symbols = append(b.symbols[:i], b.symbols[i+1:]...)

				if err := b.broker.Deregister(b, symb); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// TODO: handle errors prroperly; handle duplicates
func (b *BitmexSubscriber) subscribe(symbols []types.Symbol) error {
	b.Lock()
	defer b.Unlock()

	for _, symb := range symbols {
		i := sort.Search(len(b.symbols), func(i int) bool {
			return b.symbols[i] == symb
		})

		if i == len(b.symbols) {
			b.symbols = append(b.symbols, symb)

			if err := b.broker.Register(b, symb); err != nil {
				return err
			}
		}
	}

	return nil
}

// TODO: rewrite; ugly -- possible data races
func (b *BitmexSubscriber) Start() error {
	readerDone := b.readFromClient()

out:
	for {
		select {
		case msg := <-b.recvChan:
			if err := b.conn.WriteJSON(msg); err != nil {
				if err != ws.ErrCloseSent {
					b.setError(err)
				}

				b.Close()
				break out
			}

		case <-b.done:
			break out
		}
	}

	// TODO: handle the errors
	b.conn.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.CloseNormalClosure, ""))
	b.conn.Close()
	<-readerDone

	return b.getError()
}

// TODO: handle errors properly
func (b *BitmexSubscriber) readFromClient() <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			var req types.Request
			err := b.conn.ReadJSON(&req)

			if err != nil {
				if !ws.IsCloseError(err, ws.CloseNormalClosure) {
					b.setError(err)
				}
				return
			}

			// TODO: add some data validation
			switch req.Action {
			case "subscribe":
				b.subscribe(req.Symbols)
			case "unsubscribe":
				b.unsubscribe(req.Symbols)
			}
		}
	}()

	return done
}

func (b *BitmexSubscriber) Close() error {
	b.Do(func() {
		b.unsubscribe(b.symbols)
		close(b.done)
	})

	return nil
}
