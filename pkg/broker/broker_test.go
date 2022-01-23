package broker

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/meddion/gorilla-ws/pkg/types"
)

var _ types.Broker = &BitmexBroker{}

type MockSubscriber struct {
	types.Subscriber
	id string
}

func (m *MockSubscriber) GetID() string {
	return m.id
}

// TODO: Add more "table tests"
func TestBroker(t *testing.T) {
	broker := NewBitmexBroker()
	symbols := []types.Symbol{types.EOSUSDT, types.XBTUSDT}

	checkNumOfSubscriptions := func(t *testing.T, subID string, expected int, exists bool) {
		broker.RLock()
		defer broker.RUnlock()

		subscriber, ok := broker.subscribers[subID]
		if !ok {
			if !exists {
				return
			}
			t.Errorf("subscriber wasn't found")
		}

		if subscriber.count != expected {
			t.Errorf("number of subscriptions don't match; Got: %d; expected: %d",
				subscriber.count, expected)
		}
	}

	for i := 0; i < 20; i++ {
		t.Run(fmt.Sprintf("Sub #%d", i), func(t *testing.T) {
			sub := &MockSubscriber{id: uuid.NewString()}

			for _, symbol := range symbols {
				if err := broker.Register(sub, symbol); err != nil {
					t.Errorf("Unexpected error: %+v\n", err)
				}
			}

			checkNumOfSubscriptions(t, sub.GetID(), len(symbols), true)

			if err := broker.Register(sub, symbols[0]); err != ErrorAlreadySubscribed {
				t.Errorf("Expected error: %+v; Got: %+v\n", ErrorAlreadySubscribed, err)
			}

			if err := broker.Deregister(sub, symbols[0]); err != nil {
				t.Errorf("Unexpected error: %+v\n", err)
			}

			checkNumOfSubscriptions(t, sub.GetID(), len(symbols)-1, true)

			if err := broker.Deregister(sub, symbols[0]); err != ErrorNotFoundSubscriber {
				t.Errorf("Expected error: %+v; Got: %+v\n", ErrorNotFoundSubscriber, err)
			}

			if err := broker.Register(sub, symbols[0]); err != nil {
				t.Errorf("Unexpected error: %+v\n", err)
			}

			for _, symbol := range symbols {
				if err := broker.Deregister(sub, symbol); err != nil {
					t.Errorf("Unexpected error: %+v\n", err)
				}
			}

			checkNumOfSubscriptions(t, sub.GetID(), 0, false)
		})
	}
}
