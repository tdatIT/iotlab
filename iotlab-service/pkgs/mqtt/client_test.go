package mqtt

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/config"
)

const testTimeout = time.Second

// --- fake paho.Token ---

type fakeToken struct {
	err      error
	timedOut bool // when true WaitTimeout reports a timeout (returns false)
}

func (t *fakeToken) Wait() bool                     { return true }
func (t *fakeToken) WaitTimeout(time.Duration) bool { return !t.timedOut }
func (t *fakeToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
func (t *fakeToken) Error() error { return t.err }

// --- fake paho.Client (only the methods the wrapper uses do anything) ---

type pubRecord struct {
	topic    string
	qos      byte
	retained bool
	payload  []byte
}

type fakePaho struct {
	mu             sync.Mutex
	connected      bool
	connOpen       bool
	pubToken       *fakeToken
	subToken       *fakeToken
	unsubToken     *fakeToken
	subscribeCount map[string]int
	published      []pubRecord
	unsubscribed   []string
	disconnects    int
	handlers       map[string]pahomqtt.MessageHandler
}

func newFakePaho() *fakePaho {
	return &fakePaho{
		subscribeCount: map[string]int{},
		handlers:       map[string]pahomqtt.MessageHandler{},
	}
}

func (f *fakePaho) IsConnected() bool       { return f.connected }
func (f *fakePaho) IsConnectionOpen() bool  { return f.connOpen }
func (f *fakePaho) Connect() pahomqtt.Token { return &fakeToken{} }
func (f *fakePaho) Disconnect(uint) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.disconnects++
}

func (f *fakePaho) Publish(topic string, qos byte, retained bool, payload any) pahomqtt.Token {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, _ := payload.([]byte)
	f.published = append(f.published, pubRecord{topic: topic, qos: qos, retained: retained, payload: b})
	if f.pubToken != nil {
		return f.pubToken
	}
	return &fakeToken{}
}

func (f *fakePaho) Subscribe(topic string, _ byte, h pahomqtt.MessageHandler) pahomqtt.Token {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.subscribeCount[topic]++
	f.handlers[topic] = h
	if f.subToken != nil {
		return f.subToken
	}
	return &fakeToken{}
}

func (f *fakePaho) SubscribeMultiple(map[string]byte, pahomqtt.MessageHandler) pahomqtt.Token {
	return &fakeToken{}
}

func (f *fakePaho) Unsubscribe(topics ...string) pahomqtt.Token {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unsubscribed = append(f.unsubscribed, topics...)
	if f.unsubToken != nil {
		return f.unsubToken
	}
	return &fakeToken{}
}

func (f *fakePaho) AddRoute(string, pahomqtt.MessageHandler) {}
func (f *fakePaho) OptionsReader() pahomqtt.ClientOptionsReader {
	return pahomqtt.ClientOptionsReader{}
}

var _ pahomqtt.Client = (*fakePaho)(nil)

func noopHandler(string, []byte) {}

func (f *fakePaho) subCount(topic string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.subscribeCount[topic]
}

func (c *client) hasSub(topic string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.subs[topic]
	return ok
}

func TestClient_Publish(t *testing.T) {
	t.Run("success records the message", func(t *testing.T) {
		paho := newFakePaho()
		c := newClient(paho, testTimeout)

		require.NoError(t, c.Publish("events/x", 1, false, []byte(`{"a":1}`)))
		require.Len(t, paho.published, 1)
		assert.Equal(t, "events/x", paho.published[0].topic)
		assert.Equal(t, byte(1), paho.published[0].qos)
	})

	t.Run("timeout surfaces as error", func(t *testing.T) {
		paho := newFakePaho()
		paho.pubToken = &fakeToken{timedOut: true}
		c := newClient(paho, testTimeout)

		err := c.Publish("events/x", 1, false, []byte(`{}`))
		require.ErrorContains(t, err, "timeout")
	})

	t.Run("broker error surfaces", func(t *testing.T) {
		paho := newFakePaho()
		paho.pubToken = &fakeToken{err: errors.New("broker down")}
		c := newClient(paho, testTimeout)

		err := c.Publish("events/x", 1, false, []byte(`{}`))
		require.ErrorContains(t, err, "broker down")
	})
}

func TestClient_Subscribe(t *testing.T) {
	t.Run("connected: subscribes and registers", func(t *testing.T) {
		paho := newFakePaho()
		paho.connected = true
		c := newClient(paho, testTimeout)

		require.NoError(t, c.Subscribe("events/user", 1, noopHandler))
		assert.Equal(t, 1, paho.subCount("events/user"))
		assert.True(t, c.hasSub("events/user"))
	})

	t.Run("not connected: registers, does not subscribe, does not error", func(t *testing.T) {
		paho := newFakePaho() // connected == false
		c := newClient(paho, testTimeout)

		// Must NOT return an error — otherwise the worker's errgroup tears down every worker when the broker is down at startup.
		require.NoError(t, c.Subscribe("events/user", 1, noopHandler))
		assert.Equal(t, 0, paho.subCount("events/user"))
		assert.True(t, c.hasSub("events/user"), "registered for resubscribe on connect")
	})
}

// The key robustness guarantee: paho v1 does not resubscribe on reconnect, so the wrapper must re-issue every registered
// subscription itself.
func TestClient_ResubscribeAll(t *testing.T) {
	paho := newFakePaho()
	paho.connected = true
	c := newClient(paho, testTimeout)

	require.NoError(t, c.Subscribe("events/user", 1, noopHandler))
	require.NoError(t, c.Subscribe("events/order", 2, noopHandler))

	// Simulate a reconnect: forget the initial subscribe calls.
	paho.mu.Lock()
	paho.subscribeCount = map[string]int{}
	paho.mu.Unlock()

	c.resubscribeAll(paho)

	assert.Equal(t, 1, paho.subCount("events/user"))
	assert.Equal(t, 1, paho.subCount("events/order"))
}

func TestClient_Unsubscribe(t *testing.T) {
	paho := newFakePaho()
	paho.connected = true
	c := newClient(paho, testTimeout)
	require.NoError(t, c.Subscribe("events/user", 1, noopHandler))

	require.NoError(t, c.Unsubscribe("events/user"))
	assert.Contains(t, paho.unsubscribed, "events/user")
	assert.False(t, c.hasSub("events/user"), "unsubscribe removes the registration")
}

func TestClient_Ping(t *testing.T) {
	paho := newFakePaho()
	c := newClient(paho, testTimeout)

	require.Error(t, c.Ping(context.Background())) // connection not open

	paho.connOpen = true
	require.NoError(t, c.Ping(context.Background()))
}

// --- fake paho.Message, so the delivered payload can be asserted ---

type fakeMessage struct {
	topic   string
	payload []byte
}

func (m fakeMessage) Duplicate() bool   { return false }
func (m fakeMessage) Qos() byte         { return 1 }
func (m fakeMessage) Retained() bool    { return false }
func (m fakeMessage) Topic() string     { return m.topic }
func (m fakeMessage) MessageID() uint16 { return 1 }
func (m fakeMessage) Payload() []byte   { return m.payload }
func (m fakeMessage) Ack()              {}

var _ pahomqtt.Message = fakeMessage{}

func TestClient_Subscribe_deliversTopicAndPayloadToTheHandler(t *testing.T) {
	paho := newFakePaho()
	paho.connected = true
	c := newClient(paho, testTimeout)

	var (
		gotTopic   string
		gotPayload []byte
	)
	require.NoError(t, c.Subscribe("events/user", 1, func(topic string, payload []byte) {
		gotTopic, gotPayload = topic, payload
	}))

	paho.mu.Lock()
	h := paho.handlers["events/user"]
	paho.mu.Unlock()
	require.NotNil(t, h)
	h(paho, fakeMessage{topic: "events/user", payload: []byte(`{"a":1}`)})

	assert.Equal(t, "events/user", gotTopic)
	assert.JSONEq(t, `{"a":1}`, string(gotPayload))
}

func TestClient_Subscribe_brokerFailure_keepsTheRegistrationAndDoesNotError(t *testing.T) {
	tests := map[string]*fakeToken{
		"subscribe times out":  {timedOut: true},
		"broker returns error": {err: errors.New("broker down")},
	}

	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			paho := newFakePaho()
			paho.connected = true
			paho.subToken = token
			c := newClient(paho, testTimeout)

			// Never an error: an errgroup-driven worker must not tear the service down because the broker hiccuped.
			require.NoError(t, c.Subscribe("events/user", 1, noopHandler))
			assert.True(t, c.hasSub("events/user"), "kept for the next reconnect")
		})
	}
}

func TestClient_ResubscribeAll_brokerFailure_keepsGoingWithTheOtherTopics(t *testing.T) {
	paho := newFakePaho()
	paho.connected = true
	c := newClient(paho, testTimeout)
	require.NoError(t, c.Subscribe("events/user", 1, noopHandler))
	require.NoError(t, c.Subscribe("events/order", 1, noopHandler))

	paho.mu.Lock()
	paho.subscribeCount = map[string]int{}
	paho.subToken = &fakeToken{err: errors.New("broker down")}
	paho.mu.Unlock()

	assert.NotPanics(t, func() { c.resubscribeAll(paho) })
	assert.Equal(t, 1, paho.subCount("events/user"))
	assert.Equal(t, 1, paho.subCount("events/order"))
}

func TestClient_Unsubscribe_brokerFailure_surfacesAsError(t *testing.T) {
	tests := map[string]*fakeToken{
		"unsubscribe times out": {timedOut: true},
		"broker returns error":  {err: errors.New("broker down")},
	}

	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			paho := newFakePaho()
			paho.connected = true
			paho.unsubToken = token
			c := newClient(paho, testTimeout)
			require.NoError(t, c.Subscribe("events/user", 1, noopHandler))

			require.Error(t, c.Unsubscribe("events/user"))
			assert.False(t, c.hasSub("events/user"), "the registration is dropped regardless")
		})
	}
}

func TestClient_IsConnected_reflectsThePahoState(t *testing.T) {
	paho := newFakePaho()
	c := newClient(paho, testTimeout)

	assert.False(t, c.IsConnected())

	paho.connected = true
	assert.True(t, c.IsConnected())
}

func TestClient_Disconnect_closesThePahoConnection(t *testing.T) {
	paho := newFakePaho()
	c := newClient(paho, testTimeout)

	c.Disconnect()

	paho.mu.Lock()
	defer paho.mu.Unlock()
	assert.Equal(t, 1, paho.disconnects)
}

func TestNewMQTTClient_unreachableBroker_startsDegradedInsteadOfFailing(t *testing.T) {
	c, err := NewMQTTClient(&config.AppConfig{MQTT: config.MQTT{
		Broker:               "tcp://127.0.0.1:1",
		ClientID:             "test-client",
		ConnectTimeout:       50 * time.Millisecond,
		KeepAlive:            time.Second,
		PublishTimeout:       50 * time.Millisecond,
		ConnectRetryInterval: time.Hour, // keep the background retry out of the test
	}})

	require.NoError(t, err, "an unreachable broker must not stop the service from booting")
	require.NotNil(t, c)
	t.Cleanup(c.Disconnect)

	// Nothing is actually open: Ping is what readiness reports on.
	assert.Error(t, c.Ping(context.Background()))
}
