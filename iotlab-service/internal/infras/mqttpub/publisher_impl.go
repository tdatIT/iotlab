package mqttpub

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log/slog"

	mqttclient "github.com/tdatIT/iotlab-service/pkgs/mqtt"
)

const (
	defaultQoS      byte = 1
	defaultRetained      = false
)

type publisher struct {
	client mqttclient.Client
}

// NewPublisher returns a Publisher that marshals payloads as JSON and sends them over the given MQTT client.
func NewPublisher(client mqttclient.Client) Publisher {
	return &publisher{client: client}
}

func (p *publisher) Publish(_ context.Context, topic string, payload any) error {
	if payload == nil {
		return nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("mqtt marshal failed", slog.String("topic", topic), slog.String("error", err.Error()))
		return fmt.Errorf("mqtt marshal for %s: %w", topic, err)
	}

	if err := p.client.Publish(topic, defaultQoS, defaultRetained, data); err != nil {
		slog.Error("mqtt publish failed", slog.String("topic", topic), slog.String("error", err.Error()))
		return fmt.Errorf("mqtt publish to %s: %w", topic, err)
	}

	return nil
}
