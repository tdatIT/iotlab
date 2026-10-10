package mqttpub_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/internal/infras/mqttpub"
	mqttmock "github.com/tdatIT/iotlab-service/pkgs/mqtt/mock"
)

type samplePayload struct {
	Action string `json:"action"`
	UserID uint   `json:"user_id"`
}

func TestPublisher_Publish(t *testing.T) {
	ctx := context.Background()
	const topic = "events/user/created"

	t.Run("nil payload is a no-op", func(t *testing.T) {
		client := mqttmock.NewMockClient(t)
		pub := mqttpub.NewPublisher(client)

		require.NoError(t, pub.Publish(ctx, topic, nil))
		client.AssertNotCalled(t, "Publish")
	})

	t.Run("marshals and publishes with qos=1 retained=false", func(t *testing.T) {
		client := mqttmock.NewMockClient(t)
		payload := samplePayload{Action: "created", UserID: 1}
		want, err := json.Marshal(payload)
		require.NoError(t, err)

		client.EXPECT().
			Publish(topic, byte(1), false, mock.MatchedBy(func(b []byte) bool {
				return string(b) == string(want)
			})).
			Return(nil).
			Once()

		pub := mqttpub.NewPublisher(client)
		require.NoError(t, pub.Publish(ctx, topic, payload))
	})

	t.Run("client error is wrapped", func(t *testing.T) {
		client := mqttmock.NewMockClient(t)
		client.EXPECT().
			Publish(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(errors.New("broker down")).
			Once()

		pub := mqttpub.NewPublisher(client)
		err := pub.Publish(ctx, topic, samplePayload{Action: "created", UserID: 2})
		require.Error(t, err)
		assert.ErrorContains(t, err, "broker down")
	})
}

func TestPublisher_Publish_unmarshalablePayload_returnsAMarshalError(t *testing.T) {
	client := mqttmock.NewMockClient(t)

	err := mqttpub.NewPublisher(client).Publish(context.Background(), "events/x", make(chan int))

	require.Error(t, err)
	require.ErrorContains(t, err, "mqtt marshal for events/x")
	client.AssertNotCalled(t, "Publish")
}
