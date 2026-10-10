package probe_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/pkgs/probe"
)

type probeBody struct {
	Status  string            `json:"status"`
	Details map[string]string `json:"details"`
}

func call(t *testing.T, p *probe.Probe) (*httptest.ResponseRecorder, probeBody) {
	t.Helper()

	rec := httptest.NewRecorder()
	p.Handler()(rec, httptest.NewRequest(http.MethodGet, "/readiness", nil))

	var body probeBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "body: %s", rec.Body.String())
	return rec, body
}

func TestProbe_noCheckers_isHealthy(t *testing.T) {
	rec, body := call(t, probe.New(time.Second))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", body.Status)
	assert.Empty(t, body.Details)
}

func TestProbe_allCheckersPass_reportsOkPerChecker(t *testing.T) {
	p := probe.New(time.Second).
		Register("db", probe.CheckerFunc(func(context.Context) error { return nil })).
		Register("redis", probe.CheckerFunc(func(context.Context) error { return nil }))

	rec, body := call(t, p)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", body.Status)
	assert.Equal(t, map[string]string{"db": "ok", "redis": "ok"}, body.Details)
}

func TestProbe_oneCheckerFails_reportsDegradedWithTheReason(t *testing.T) {
	p := probe.New(time.Second).
		Register("db", probe.CheckerFunc(func(context.Context) error { return nil })).
		Register("redis", probe.CheckerFunc(func(context.Context) error { return errors.New("redis down") }))

	rec, body := call(t, p)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "degraded", body.Status)
	assert.Equal(t, "ok", body.Details["db"])
	assert.Equal(t, "redis down", body.Details["redis"])
}

func TestProbe_slowChecker_isBoundedByTheProbeTimeout(t *testing.T) {
	p := probe.New(20*time.Millisecond).
		Register("slow", probe.CheckerFunc(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}))

	rec, body := call(t, p)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, context.DeadlineExceeded.Error(), body.Details["slow"])
}

func TestProbe_reRegisteringAName_replacesTheChecker(t *testing.T) {
	p := probe.New(time.Second).
		Register("db", probe.CheckerFunc(func(context.Context) error { return errors.New("old") })).
		Register("db", probe.CheckerFunc(func(context.Context) error { return nil }))

	rec, body := call(t, p)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, body.Details, 1)
}

func TestCheckerFunc_Check_callsTheFunction(t *testing.T) {
	want := errors.New("boom")
	assert.ErrorIs(t, probe.CheckerFunc(func(context.Context) error { return want }).Check(context.Background()), want)
}
