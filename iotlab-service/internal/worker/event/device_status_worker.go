package event

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tdatIT/iotlab-service/config"
	deviceapp "github.com/tdatIT/iotlab-service/internal/app/device"
	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	mqttclient "github.com/tdatIT/iotlab-service/pkgs/mqtt"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/validate"
)

// DeviceStatusWorker applies the retained plain-text status ("online" | "offline") published on
// `homelab/iot/devices/{device_id}/status`.
type DeviceStatusWorker struct {
	client mqttclient.Client
	app    *deviceapp.Application
	cfg    *config.WorkerConfig
}

// NewDeviceStatusWorker builds the device status worker.
func NewDeviceStatusWorker(client mqttclient.Client, app *deviceapp.Application, cfg *config.WorkerConfig) *DeviceStatusWorker {
	return &DeviceStatusWorker{client: client, app: app, cfg: cfg}
}

// Start subscribes to the status topic and blocks until ctx is cancelled.
func (w *DeviceStatusWorker) Start(ctx context.Context) error {
	err := w.client.Subscribe(w.cfg.Topic, w.cfg.QoS, func(topic string, payload []byte) {
		go w.handle(ctx, topic, payload) // non-blocking: never hold the MQTT message loop
	})
	if err != nil {
		return fmt.Errorf("device status worker: subscribe %s: %w", w.cfg.Topic, err)
	}
	<-ctx.Done()
	return nil
}

func (w *DeviceStatusWorker) handle(ctx context.Context, topic string, raw []byte) {
	req := &devicedtos.UpdateDeviceStatusReq{
		DeviceID: deviceIDFromTopic(topic),
		Status:   strings.TrimSpace(string(raw)),
	}
	if err := validate.GetValidator().Validate(req); err != nil {
		slog.Warn("device status: invalid payload", slog.String("topic", topic), slog.String("error", err.Error()))
		return
	}

	// The app layer logs the cause of a failure; the message is dropped (QoS 0, no redelivery).
	_ = w.app.Command.UpdateDeviceStatus.Handle(ctx, req)
}
