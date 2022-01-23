package types

type Topic string

const (
	OrderBookL2_25 Topic = "orderBookL2_25"
	Instrument     Topic = "instrument"
)

var DefaultTopics = []Topic{
	"instrument",
}

func ApplySymbolFilterToTopic(topic Topic, filter Symbol) Topic {
	return Topic(string(topic) + ":" + string(filter))
}

type Symbol string

// TODO: add more filters
const (
	XBTUSDT Symbol = "XBTUSDT"
	// TODO: this filter doesn't work; why?
	EOSUSDT Symbol = "EOSUSDT"
	LTCUSDT Symbol = "LTCUSDT"
)

var DefaultSymbols = []Symbol{
	XBTUSDT, EOSUSDT, LTCUSDT,
}

type AuthConfig struct {
	URL       string
	ApiKey    string
	ApiSecret string
	Verb      string
	Endpoint  string
}

type Request struct {
	Action  string   `json:"action" binding:"required"`
	Symbols []Symbol `json:"symbols"`
}
type Message struct {
	Symbol    Symbol  `json:"symbol"`
	Price     float64 `json:"price"`
	Timestamp string  `json:"timestamp"`
}

type WithError interface {
	GetError() error
}

type Identity interface {
	GetID() string
}

type WithClose interface {
	Close() error
}

type Broker interface {
	Identity
	WithClose
	Register(Subscriber, Symbol) error
	Deregister(Subscriber, Symbol) error
	Broadcast(Message) error
}

type Subscriber interface {
	Identity
	WithClose
	Receive(Message)
}

type Publisher interface {
	WithClose
	StartPublishing(Broker) error
}
