package subscriber

import (
	"math/rand"
	"testing"
	"time"

	"github.com/meddion/gorilla-ws/pkg/types"
)

var _ types.Subscriber = &BitmexSubscriber{}

type mockPublisher struct {
	types.Publisher
}

func (m *mockPublisher) StartPublishing(brk types.Broker) error {
	rand.Seed(42)
	const TOTAL = 25

	for i := 0; i < TOTAL; i++ {
		msg := types.Message{
			Symbol:    types.DefaultSymbols[i%len(types.DefaultSymbols)],
			Price:     2000.0*rand.Float64() + 1000.0,
			Timestamp: time.Now().String(),
		}
		time.Sleep(time.Millisecond * 200)

		brk.Broadcast(msg)
	}

	return nil
}

// TODO: complete the test
func TestSubscriber_Success(t *testing.T) {

}
