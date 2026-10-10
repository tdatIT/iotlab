//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// envelope mirrors response.BaseRes with Data left raw so each test decodes its own payload type.
type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// call sends a request to the live server and decodes the response envelope. body may be nil.
func call(t *testing.T, method, path string, body any) (int, envelope) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, srv.URL+path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()

	var env envelope
	require.NoError(t, json.NewDecoder(res.Body).Decode(&env))
	return res.StatusCode, env
}

// decodeData unmarshals the envelope payload into out.
func decodeData(t *testing.T, env envelope, out any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(env.Data, out))
}
