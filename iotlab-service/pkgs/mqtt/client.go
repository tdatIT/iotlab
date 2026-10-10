// Package mqtt wraps the paho MQTT client, keeping paho types out of the publishers and workers that use it.
package mqtt

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/tdatIT/iotlab-service/config"
)

// MessageHandler receives the topic and raw payload of an incoming message. It keeps paho types out of the callers (publishers
// and workers).
type MessageHandler func(topic string, payload []byte)

// Client is the MQTT abstraction used by publishers and workers. A single Client wraps ONE paho connection (one ClientID) and
// multiplexes many publishers and many subscribers over it.
type Client interface {
	Publish(topic string, qos byte, retained bool, payload []byte) error
	Subscribe(topic string, qos byte, handler MessageHandler) error
	Unsubscribe(topic string) error
	Ping(ctx context.Context) error
	IsConnected() bool
	Disconnect()
}

// subscription is one registered (topic, qos, handler) tuple, kept so every subscription can be re-issued whenever paho
// reconnects — paho v1 does NOT resubscribe automatically.
type subscription struct {
	qos     byte
	handler MessageHandler
}

type client struct {
	paho           pahomqtt.Client
	publishTimeout time.Duration

	mu   sync.RWMutex
	subs map[string]subscription // topic -> subscription
}

// NewMQTTClient builds a shared MQTT client. It never fails because the broker is unreachable: connection is retried in the
// background and the service starts degraded (readiness reports it, workers subscribe once connected).
func NewMQTTClient(cfg *config.AppConfig) (Client, error) {
	c := &client{
		publishTimeout: cfg.MQTT.PublishTimeout,
		subs:           make(map[string]subscription),
	}

	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(cfg.MQTT.Broker)
	opts.SetClientID(cfg.MQTT.ClientID)
	opts.SetUsername(cfg.MQTT.Username)
	opts.SetPassword(cfg.MQTT.Password)
	opts.SetConnectTimeout(cfg.MQTT.ConnectTimeout)
	opts.SetKeepAlive(cfg.MQTT.KeepAlive)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(cfg.MQTT.ConnectRetryInterval)

	// Fires on the first connect AND every reconnect. Re-issues every registered subscription so subscribers
	// survive broker restarts.
	opts.SetOnConnectHandler(func(pc pahomqtt.Client) {
		slog.Info("mqtt client connected", slog.String("broker", cfg.MQTT.Broker))
		c.resubscribeAll(pc)
	})
	opts.SetConnectionLostHandler(func(_ pahomqtt.Client, err error) {
		slog.Warn("mqtt connection lost", slog.String("error", err.Error()))
	})
	opts.SetReconnectingHandler(func(_ pahomqtt.Client, _ *pahomqtt.ClientOptions) {
		slog.Info("mqtt reconnecting...")
	})

	c.paho = pahomqtt.NewClient(opts)

	// Wait briefly for the first connection; if it does not complete in time we start degraded and let ConnectRetry keep trying in
	// the background.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MQTT.ConnectTimeout)
	defer cancel()
	token := c.paho.Connect()
	select {
	case <-token.Done():
		if err := token.Error(); err != nil {
			slog.Warn("mqtt initial connect error; starting degraded, retrying in background",
				slog.String("error", err.Error()))
		}
	case <-ctx.Done():
		slog.Warn("mqtt not connected within timeout; starting degraded, retrying in background",
			slog.String("timeout", cfg.MQTT.ConnectTimeout.String()))
	}

	return c, nil
}

// newClient is the test seam: it builds the wrapper around an already-created paho client so tests can inject a fake without a
// real broker.
func newClient(paho pahomqtt.Client, publishTimeout time.Duration) *client {
	return &client{
		paho:           paho,
		publishTimeout: publishTimeout,
		subs:           make(map[string]subscription),
	}
}

func (c *client) Publish(topic string, qos byte, retained bool, payload []byte) error {
	token := c.paho.Publish(topic, qos, retained, payload)
	if !token.WaitTimeout(c.publishTimeout) {
		return fmt.Errorf("mqtt publish %s: timeout after %s", topic, c.publishTimeout)
	}
	return token.Error()
}

// Subscribe registers the (topic, qos, handler) tuple first, then subscribes on the live connection. When the client is not
// connected yet it returns nil (the tuple is applied on the next connect) so a broker outage never tears down a worker via the
// errgroup.
func (c *client) Subscribe(topic string, qos byte, handler MessageHandler) error {
	c.mu.Lock()
	c.subs[topic] = subscription{qos: qos, handler: handler}
	c.mu.Unlock()

	if !c.paho.IsConnected() {
		slog.Warn("mqtt not connected; subscription registered, will apply on connect",
			slog.String("topic", topic))
		return nil
	}
	if err := c.subscribe(c.paho, topic, qos, handler); err != nil {
		slog.Error("mqtt subscribe failed; will retry on reconnect",
			slog.String("topic", topic), slog.String("error", err.Error()))
		return nil
	}
	return nil
}

// subscribe wires paho's (Client, Message) callback down to our []byte handler.
func (c *client) subscribe(pc pahomqtt.Client, topic string, qos byte, h MessageHandler) error {
	token := pc.Subscribe(topic, qos, func(_ pahomqtt.Client, m pahomqtt.Message) {
		h(m.Topic(), m.Payload())
	})
	if !token.WaitTimeout(c.publishTimeout) {
		return fmt.Errorf("subscribe timeout")
	}
	return token.Error()
}

// resubscribeAll re-issues every registered subscription. It snapshots under the read lock and never awaits a network token while
// holding the lock.
func (c *client) resubscribeAll(pc pahomqtt.Client) {
	c.mu.RLock()
	snapshot := make(map[string]subscription, len(c.subs))
	for t, s := range c.subs {
		snapshot[t] = s
	}
	c.mu.RUnlock()

	for topic, s := range snapshot {
		if err := c.subscribe(pc, topic, s.qos, s.handler); err != nil {
			slog.Error("mqtt resubscribe failed",
				slog.String("topic", topic), slog.String("error", err.Error()))
			continue
		}
		slog.Info("mqtt resubscribed", slog.String("topic", topic))
	}
}

func (c *client) Unsubscribe(topic string) error {
	c.mu.Lock()
	delete(c.subs, topic)
	c.mu.Unlock()

	token := c.paho.Unsubscribe(topic)
	if !token.WaitTimeout(c.publishTimeout) {
		return fmt.Errorf("mqtt unsubscribe %s: timeout", topic)
	}
	return token.Error()
}

func (c *client) Ping(_ context.Context) error {
	if !c.paho.IsConnectionOpen() {
		return fmt.Errorf("mqtt: connection not open")
	}
	return nil
}

func (c *client) IsConnected() bool { return c.paho.IsConnected() }

func (c *client) Disconnect() { c.paho.Disconnect(250) }
