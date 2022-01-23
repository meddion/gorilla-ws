package publisher

import (
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/meddion/gorilla-ws/pkg/types"
)

var _ types.Publisher = &BitmexPublisher{}

type MockBroker struct {
	types.Broker
	broadcast func(msg types.Message)
}

func (m *MockBroker) Broadcast(msg types.Message) error {
	m.broadcast(msg)
	return nil
}

func TestBitmexPublisher_Success(t *testing.T) {
	// TODO: test for other topics
	topics := []types.Topic{types.Instrument}

	// TODO: make it right
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("On reading from .env: %+v\n", err)
	}

	authCfg := types.AuthConfig{
		URL:       os.Getenv("URL"),
		ApiKey:    os.Getenv("API_KEY"),
		ApiSecret: os.Getenv("API_SECRET"),
		Endpoint:  os.Getenv("ENDPOINT"),
		Verb:      os.Getenv("VERB"),
	}
	url := authCfg.URL + authCfg.Endpoint

	c := NewBitmexPublisher(url, &authCfg, topics)

	broker := &MockBroker{}
	counter := 0
	broker.broadcast = func(msg types.Message) {
		if msg.Symbol != "" && msg.Timestamp != "" {
			counter++
		}
	}

	time.AfterFunc(time.Second*8, func() {
		if err := c.Close(); err != nil {
			t.Errorf("On closing a WS connection: %+v", err)
		}
	})

	if err := c.StartPublishing(broker); err != nil {
		t.Fatalf("Got unexpected error: %+v", err)
	}

	if counter < 5 {
		t.Fatalf("Got underflow of valid messages; Expected %d, got %d ", 5, counter)
	}
}

func TestGenBitmexSignature_Success(t *testing.T) {
	// Took from the python example
	apiSecret := "f9XOPLacPCZJ1dvPzN8B6Et7nMEaPGeomMSHk8Cr2zD4NfCY"
	expectedSig := "ced262fa83b7ccc7d1a38bfe594c1e2d986c0836d6bff7f7dc2e00c9382d52ef"
	expires := 1642593867
	// expires := int(time.Now().Add(time.Second * 5).Unix())

	sig, err := genBitmexSignature(apiSecret, "GET", "/realtime", expires)

	if err != nil {
		t.Fatalf("Unexpected error: %+v", err)
	}

	if expectedSig != sig {
		t.Fatalf("expectedSig != sig; sig = %s\n", sig)
	}
}
