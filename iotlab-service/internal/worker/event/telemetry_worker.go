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

// TelemetryWorker stores the health reports published on `homelab/iot/devices/{device_id}/telemetry`.
type TelemetryWorker struct {
	client mqttclient.Client
	app    *deviceapp.Application
	cfg    *config.WorkerConfig
}

// NewTelemetryWorker builds the telemetry worker.
func NewTelemetryWorker(client mqttclient.Client, app *deviceapp.Application, cfg *config.WorkerConfig) *TelemetryWorker {
	return &TelemetryWorker{client: client, app: app, cfg: cfg}
}

// Start subscribes to the telemetry topic and blocks until ctx is cancelled.
func (w *TelemetryWorker) Start(ctx context.Context) error {
	err := w.client.Subscribe(w.cfg.Topic, w.cfg.QoS, func(topic string, payload []byte) {
		go w.handle(ctx, topic, payload) // non-blocking: never hold the MQTT message loop
	})
	if err != nil {
		return fmt.Errorf("telemetry worker: subscribe %s: %w", w.cfg.Topic, err)
	}
	<-ctx.Done()
	return nil
}

func (w *TelemetryWorker) handle(ctx context.Context, topic string, raw []byte) {
	var payload msgs.TelemetryPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		slog.Warn("telemetry: malformed payload", slog.String("topic", topic), slog.String("error", err.Error()))
		return
	}

	req := &devicedtos.RecordTelemetryReq{
		DeviceID:       deviceIDFromTopic(topic),
		Time:           payload.Time,
		FW:             payload.FW,
		UptimeS:        payload.UptimeS,
		ResetReason:    payload.ResetReason,
		RSSIDbm:        payload.RSSIDbm,
		IP:             payload.IP,
		HeapFree:       payload.HeapFree,
		HeapMin:        payload.HeapMin,
		WifiReconnects: payload.WifiReconnects,
		MQTTReconnects: payload.MQTTReconnects,
		SensorErrors:   payload.SensorErrors,
	}
	if err := validate.GetValidator().Validate(req); err != nil {
		slog.Warn("telemetry: invalid payload", slog.String("topic", topic), slog.String("error", err.Error()))
		return
	}

	// The app layer logs the cause of a failure; the message is dropped (QoS 0, no redelivery).
	_ = w.app.Command.RecordTelemetry.Handle(ctx, req)
}
