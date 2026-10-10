package event

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log/slog"

	"github.com/tdatIT/iotlab-service/config"
	deviceapp "github.com/tdatIT/iotlab-service/internal/app/device"
	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	"github.com/tdatIT/iotlab-service/internal/domain/msgs"
	mqttclient "github.com/tdatIT/iotlab-service/pkgs/mqtt"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/validate"
)

// SensorWorker stores the readings published on `homelab/iot/env/{device_id}/sensor`.
type SensorWorker struct {
	client mqttclient.Client
	app    *deviceapp.Application
	cfg    *config.WorkerConfig
}

// NewSensorWorker builds the sensor worker.
func NewSensorWorker(client mqttclient.Client, app *deviceapp.Application, cfg *config.WorkerConfig) *SensorWorker {
	return &SensorWorker{client: client, app: app, cfg: cfg}
}

// Start subscribes to the sensor topic and blocks until ctx is cancelled.
func (w *SensorWorker) Start(ctx context.Context) error {
	err := w.client.Subscribe(w.cfg.Topic, w.cfg.QoS, func(topic string, payload []byte) {
		go w.handle(ctx, topic, payload) // non-blocking: never hold the MQTT message loop
	})
	if err != nil {
		return fmt.Errorf("sensor worker: subscribe %s: %w", w.cfg.Topic, err)
	}
	<-ctx.Done()
	return nil
}

func (w *SensorWorker) handle(ctx context.Context, topic string, raw []byte) {
	var payload msgs.SensorPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		slog.Warn("sensor: malformed payload", slog.String("topic", topic), slog.String("error", err.Error()))
		return
	}

	req := &devicedtos.RecordSensorReq{
		DeviceID:    deviceIDFromTopic(topic),
		TempC:       payload.Temp,
		HumidityPct: payload.Hum,
		Time:        payload.Time,
	}
	if err := validate.GetValidator().Validate(req); err != nil {
		slog.Warn("sensor: invalid payload", slog.String("topic", topic), slog.String("error", err.Error()))
		return
	}

	// The app layer logs the cause of a failure; the message is dropped (QoS 0, no redelivery).
	_ = w.app.Command.RecordSensor.Handle(ctx, req)
}
