package publisher

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
	"github.com/meddion/gorilla-ws/pkg/types"
)

type CmdOpType string

const (
	SubOp    CmdOpType = "subscribe"
	UcnsubOP CmdOpType = "unsubscribe"
	AuthOp   CmdOpType = "authKeyExpires"
)

type BitmexCmd struct {
	Op   CmdOpType     `json:"op"`
	Args []interface{} `json:"args"`
}

// TODO: extend; there is much more to it
type Data struct {
	Symbol         string  `json:"symbol"`
	BidPrice       float64 `json:"bidPrice"`
	MidPrice       float64 `json:"midPrice"`
	AskPrice       float64 `json:"askPrice"`
	ImpactBidPrice float64 `json:"impactBidPrice"`
	ImpactAskPrice float64 `json:"impactAskPrice"`
	Timestamp      string  `json:"timestamp"`
}

type BitmexAPIResponse struct {
	// TODO: extend
	Table  string `json:"table"`
	Action string `json:"action"`
	Data   []Data `json:"data"`
}

func (b *BitmexAPIResponse) ToMessage() types.Message {
	if len(b.Data) == 0 {
		return types.Message{}
	}

	symb := strings.Replace(b.Data[0].Symbol, ".", "", 1)
	return types.Message{
		Symbol:    types.Symbol(symb),
		Price:     b.Data[0].MidPrice,
		Timestamp: b.Data[0].Timestamp,
	}
}

type Response struct {
	resp *BitmexAPIResponse
	error
}

type BitmexPublisher struct {
	sync.RWMutex
	done      chan struct{}
	conn      *ws.Conn
	authCfg   *types.AuthConfig
	topics    []types.Topic
	sourceURL string
}

func NewBitmexPublisher(sourceURL string, authCfg *types.AuthConfig, topics []types.Topic) *BitmexPublisher {
	return &BitmexPublisher{
		sourceURL: sourceURL,
		authCfg:   authCfg,
		topics:    topics,
		done:      make(chan struct{}),
	}
}

func (b *BitmexPublisher) Close() error {
	close(b.done)
	// TODO: init the closing handshake
	return b.conn.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.CloseNormalClosure, ""))
}

func (b *BitmexPublisher) StartPublishing(broker types.Broker) error {
	ctx, ctxCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxCancel()
	msgChan, err := b.startPulling(ctx)

	if err != nil {
		return err
	}

	for {
		apiMsg, ok := <-msgChan
		if !ok {
			break
		}

		if apiMsg.error != nil {
			return apiMsg.error
		}

		// Assumming the broker is always active
		go broker.Broadcast(apiMsg.resp.ToMessage())
	}

	return nil
}

func (b *BitmexPublisher) startPulling(ctx context.Context) (<-chan Response, error) {
	// TODO: change to a custom dialer; maybe auth through headers?
	// TODO: Set read deadline, rate limiting, etc
	var err error
	b.conn, _, err = ws.DefaultDialer.DialContext(ctx, b.sourceURL, nil)
	if err != nil {
		return nil, err
	}

	msg := BitmexCmd{Op: SubOp}
	topics := b.topics
	if len(b.topics) == 0 {
		topics = types.DefaultTopics
	}

	for _, topic := range topics {
		msg.Args = append(msg.Args, topic)
	}

	if b.authCfg != nil {
		// TODO: add verbose messages
		if err := b.auth(); err != nil {
			b.Close()
			return nil, err
		}

	}

	if err := b.conn.WriteJSON(msg); err != nil {
		b.Close()
		return nil, err
	}

	sendChan := make(chan Response)

	// TODO: somewhat complex; maybe simplifty?
	go func() {
		defer b.conn.Close()
		for {
			var resp BitmexAPIResponse
			if err := b.conn.ReadJSON(&resp); err != nil {
				// TODO: add more to the recovery on error
				if !ws.IsCloseError(err, ws.CloseNormalClosure) {
					select {
					case sendChan <- Response{nil, err}:
					case <-time.After(5 * time.Second):
					}
				}

				close(sendChan)

				break
			}

			select {
			case sendChan <- Response{&resp, nil}:
			case <-b.done:
			}
		}

	}()
	return sendChan, nil
}

func (b *BitmexPublisher) auth() error {
	// TODO: what the expires is used for?
	expires := int(time.Now().Add(time.Second * 5).Unix())

	signature, err := genBitmexSignature(b.authCfg.ApiSecret, b.authCfg.Verb, b.authCfg.Endpoint, expires)
	if err != nil {
		return err
	}

	cmd := BitmexCmd{Op: AuthOp, Args: []interface{}{
		b.authCfg.ApiKey, expires, signature,
	}}
	// TODO: add custom errors
	return b.conn.WriteJSON(cmd)
}

// TODO: maybe group with the current AuthCfg
func genBitmexSignature(apiSecret, verb, endpoint string, expires int) (string, error) {
	nonce := strconv.Itoa(expires)
	msg := []byte(verb + endpoint + nonce)

	hash := hmac.New(sha256.New, []byte(apiSecret))
	hash.Write(msg)

	return hex.EncodeToString(hash.Sum(nil)), nil
}
