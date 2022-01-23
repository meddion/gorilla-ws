package broker

import (
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/meddion/gorilla-ws/pkg/types"
)

const SUB_CAPACITY = 10

var (
	ErrorNoSubscribers      = errors.New("No subscribers to send message to")
	ErrorAlreadySubscribed  = errors.New("A Subscriber already attached to this symbol")
	ErrorNotFoundSubscriber = errors.New("A Subscriber wasn't found under the symbol")
)

type subWithCounter struct {
	sub   types.Subscriber
	count int
}

type BitmexBroker struct {
	// TODO: use Map from sync?
	sync.RWMutex
	id string
	// TODO: use 'subscribers' as an index registry for symbMap
	symbMap     map[types.Symbol][]types.Subscriber
	subscribers map[string]*subWithCounter
}

func NewBitmexBroker() *BitmexBroker {
	return &BitmexBroker{
		id:          uuid.NewString(),
		symbMap:     make(map[types.Symbol][]types.Subscriber),
		subscribers: make(map[string]*subWithCounter),
	}
}

func (b *BitmexBroker) GetID() string {
	return b.id
}

func (b *BitmexBroker) Register(newSub types.Subscriber, symbol types.Symbol) error {
	b.Lock()
	defer b.Unlock()

	newSubID := newSub.GetID()

	if _, ok := b.symbMap[symbol]; !ok {
		b.symbMap[symbol] = make([]types.Subscriber, 0, SUB_CAPACITY)
	} else {
		for _, sub := range b.symbMap[symbol] {
			if sub.GetID() == newSubID {
				return ErrorAlreadySubscribed
			}
		}
	}

	if _, ok := b.subscribers[newSubID]; !ok {
		b.subscribers[newSubID] = &subWithCounter{newSub, 0}
	}

	meta := b.subscribers[newSubID]
	meta.count += 1
	b.symbMap[symbol] = append(b.symbMap[symbol], newSub)

	return nil
}

func (b *BitmexBroker) Deregister(newSub types.Subscriber, symbol types.Symbol) error {
	b.Lock()
	defer b.Unlock()

	newSubID := newSub.GetID()

	if _, ok := b.symbMap[symbol]; !ok {
		return ErrorNotFoundSubscriber
	}

	for i, sub := range b.symbMap[symbol] {
		if sub.GetID() == newSubID {
			b.symbMap[symbol] = append(b.symbMap[symbol][:i], b.symbMap[symbol][i+1:]...)

			meta := b.subscribers[newSubID]
			meta.count -= 1
			if meta.count == 0 {
				delete(b.subscribers, newSubID)
			}

			return nil
		}
	}

	return ErrorNotFoundSubscriber
}

func (b *BitmexBroker) Broadcast(msg types.Message) error {
	b.RLock()
	defer b.RUnlock()

	if len(b.symbMap[msg.Symbol]) == 0 {
		return ErrorNoSubscribers
	}

	if subs, ok := b.symbMap[msg.Symbol]; ok {
		for _, sub := range subs {
			go sub.Receive(msg)
		}
	}

	return nil
}

// TODO: handle errors properly
func (b *BitmexBroker) Close() error {
	b.Lock()
	defer b.Unlock()

	for _, meta := range b.subscribers {
		go meta.sub.Close()
	}

	return nil
}
