package event_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/config"
	deviceapp "github.com/tdatIT/iotlab-service/internal/app/device"
	"github.com/tdatIT/iotlab-service/internal/domain/models"
	devicemock "github.com/tdatIT/iotlab-service/internal/infras/repository/device/mock"
	sensormock "github.com/tdatIT/iotlab-service/internal/infras/repository/sensor/mock"
	telemetrymock "github.com/tdatIT/iotlab-service/internal/infras/repository/telemetry/mock"
	"github.com/tdatIT/iotlab-service/internal/worker/event"
	pkgmqtt "github.com/tdatIT/iotlab-service/pkgs/mqtt"
	mqttmock "github.com/tdatIT/iotlab-service/pkgs/mqtt/mock"
)

type repos struct {
	device    *devicemock.MockRepository
	sensor    *sensormock.MockRepository
	telemetry *telemetrymock.MockRepository
}

func newApp(t *testing.T) (*deviceapp.Application, repos) {
	t.Helper()
	r := repos{
		device:    devicemock.NewMockRepository(t),
		sensor:    sensormock.NewMockRepository(t),
		telemetry: telemetrymock.NewMockRepository(t),
	}
	return deviceapp.NewDeviceApplication(r.device, r.sensor, r.telemetry), r
}

type starter interface {
	Start(ctx context.Context) error
}

// start runs w until the test ends and returns the message handler it subscribed with.
func start(t *testing.T, client *mqttmock.MockClient, w starter, topic string) pkgmqtt.MessageHandler {
	t.Helper()

	subscribed := make(chan pkgmqtt.MessageHandler, 1)
	client.EXPECT().Subscribe(topic, byte(0), mock.Anything).
		Run(func(_ string, _ byte, h pkgmqtt.MessageHandler) { subscribed <- h }).
		Return(nil).Once()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Start(ctx) }()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-done)
	})

	select {
	case h := <-subscribed:
		return h
	case <-time.After(time.Second):
		t.Fatal("worker did not subscribe")
		return nil
	}
}

// signal returns a channel closed by the returned func, used to wait for the async handler to reach the repository.
func signal() (chan struct{}, func(context.Context, *models.Device, ...string)) {
	ch := make(chan struct{})
	return ch, func(context.Context, *models.Device, ...string) { close(ch) }
}

func wait(t *testing.T, ch chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("handler did not reach the repository")
	}
}

func TestSensorWorker(t *testing.T) {
	const topic = "homelab/iot/env/+/sensor"
	cfg := &config.WorkerConfig{Topic: topic, QoS: 0}

	t.Run("valid payload is stored for the device named in the topic", func(t *testing.T) {
		client := mqttmock.NewMockClient(t)
		app, r := newApp(t)
		handle := start(t, client, event.NewSensorWorker(client, app, cfg), topic)

		done, notify := signal()
		r.sensor.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(s *models.SensorRaw) bool {
				return s.DeviceID == "esp32-333898" && s.TempC == 31.6 && s.HumidityPct == 74.5 &&
					s.Time.UnixMilli() == 1791620022386
			})).
			Return(nil).Once()
		r.device.EXPECT().Upsert(mock.Anything, mock.Anything, []string{"last_seen"}).Run(notify).Return(nil).Once()

		handle("homelab/iot/env/esp32-333898/sensor", []byte(`{"temp":31.6,"hum":74.5,"time":1791620022386}`))
		wait(t, done)
	})

	for name, payload := range map[string]string{
		"malformed json":     `{"temp":`,
		"temp out of range":  `{"temp":120,"hum":50,"time":1}`,
		"missing time field": `{"temp":20,"hum":50}`,
		"duplicate key":      `{"temp":20,"temp":21,"hum":50,"time":1}`,
	} {
		t.Run(name+" is dropped", func(t *testing.T) {
			client := mqttmock.NewMockClient(t)
			app, r := newApp(t)
			handle := start(t, client, event.NewSensorWorker(client, app, cfg), topic)

			handle("homelab/iot/env/esp-1/sensor", []byte(payload))
			time.Sleep(50 * time.Millisecond)

			r.sensor.AssertNotCalled(t, "Create")
		})
	}
}

func TestTelemetryWorker(t *testing.T) {
	const topic = "homelab/iot/devices/+/telemetry"
	cfg := &config.WorkerConfig{Topic: topic, QoS: 0}

	t.Run("valid payload is stored for the device named in the topic", func(t *testing.T) {
		client := mqttmock.NewMockClient(t)
		app, r := newApp(t)
		handle := start(t, client, event.NewTelemetryWorker(client, app, cfg), topic)

		done, notify := signal()
		r.telemetry.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(m *models.DeviceTelemetry) bool {
				return m.DeviceID == "esp32-333898" && *m.FW == "1.2.0-dht22" && *m.RSSIDbm == -48 &&
					*m.HeapFree == 226368 && *m.SensorErrors == 1 && m.Time.UnixMilli() == 1791620046520
			})).
			Return(nil).Once()
		r.device.EXPECT().Upsert(mock.Anything, mock.Anything, []string{"last_seen", "fw", "ip"}).Run(notify).Return(nil).Once()

		handle("homelab/iot/devices/esp32-333898/telemetry", []byte(`{"time":1791620046520,"fw":"1.2.0-dht22",`+
			`"uptime_s":10443,"reset_reason":"poweron","rssi_dbm":-48,"ip":"192.168.1.156","heap_free":226368,`+
			`"heap_min":143620,"wifi_reconnects":0,"mqtt_reconnects":0,"sensor_errors":1}`))
		wait(t, done)
	})

	for name, payload := range map[string]string{
		"malformed json": `not json`,
		"invalid ip":     `{"time":1,"ip":"999.1.1.1"}`,
	} {
		t.Run(name+" is dropped", func(t *testing.T) {
			client := mqttmock.NewMockClient(t)
			app, r := newApp(t)
			handle := start(t, client, event.NewTelemetryWorker(client, app, cfg), topic)

			handle("homelab/iot/devices/esp-1/telemetry", []byte(payload))
			time.Sleep(50 * time.Millisecond)

			r.telemetry.AssertNotCalled(t, "Create")
		})
	}
}

func TestDeviceStatusWorker(t *testing.T) {
	const topic = "homelab/iot/devices/+/status"
	cfg := &config.WorkerConfig{Topic: topic, QoS: 0}

	t.Run("plain-text status is applied to the device named in the topic", func(t *testing.T) {
		client := mqttmock.NewMockClient(t)
		app, r := newApp(t)
		handle := start(t, client, event.NewDeviceStatusWorker(client, app, cfg), topic)

		done, notify := signal()
		r.device.EXPECT().FindByID(mock.Anything, "esp32-333898").
			Return(&models.Device{Status: models.DeviceStatusOffline}, nil).Once()
		r.device.EXPECT().
			Upsert(mock.Anything, mock.MatchedBy(func(d *models.Device) bool { return d.Status == models.DeviceStatusOnline }),
				[]string{"status", "status_changed_at", "last_seen"}).
			Run(notify).Return(nil).Once()

		handle("homelab/iot/devices/esp32-333898/status", []byte("online\n"))
		wait(t, done)
	})

	t.Run("unknown status is dropped", func(t *testing.T) {
		client := mqttmock.NewMockClient(t)
		app, r := newApp(t)
		handle := start(t, client, event.NewDeviceStatusWorker(client, app, cfg), topic)

		handle("homelab/iot/devices/esp-1/status", []byte("rebooting"))
		time.Sleep(50 * time.Millisecond)

		r.device.AssertNotCalled(t, "FindByID")
	})
}

func TestWorkers_subscribeError_isReturned(t *testing.T) {
	app, _ := newApp(t)
	cfg := &config.WorkerConfig{Topic: "t", QoS: 0}

	for name, newWorker := range map[string]func(pkgmqtt.Client) starter{
		"sensor":    func(c pkgmqtt.Client) starter { return event.NewSensorWorker(c, app, cfg) },
		"telemetry": func(c pkgmqtt.Client) starter { return event.NewTelemetryWorker(c, app, cfg) },
		"status":    func(c pkgmqtt.Client) starter { return event.NewDeviceStatusWorker(c, app, cfg) },
	} {
		t.Run(name, func(t *testing.T) {
			client := mqttmock.NewMockClient(t)
			client.EXPECT().Subscribe("t", byte(0), mock.Anything).Return(errors.New("broker down")).Once()

			err := newWorker(client).Start(context.Background())

			assert.ErrorContains(t, err, "broker down")
		})
	}
}
