// Package event holds the MQTT child workers. Each worker decodes one topic family, validates the decoded request and hands it
// to the app layer.
package event

import "path"

// deviceIDFromTopic returns the {device_id} segment of `.../{device_id}/<leaf>` topics,
// e.g. homelab/iot/env/esp32-333898/sensor -> esp32-333898.
func deviceIDFromTopic(topic string) string {
	return path.Base(path.Dir(topic))
}
