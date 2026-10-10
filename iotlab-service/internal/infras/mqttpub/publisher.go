// Package mqttpub publishes domain events to the MQTT broker.
package mqttpub

import "context"

// Publisher sends messages to an MQTT broker. One shared Publisher is injected into every app command that needs to emit events
// — many publishers, one underlying connection.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}
