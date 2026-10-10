// Package probe exposes a framework-agnostic readiness endpoint that runs every registered health checker concurrently.
package probe

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"sync"
	"time"
)

// Checker is implemented by any component that can report its health.
type Checker interface {
	Check(ctx context.Context) error
}

// CheckerFunc adapts a function to the Checker interface.
type CheckerFunc func(ctx context.Context) error

// Check calls f.
func (f CheckerFunc) Check(ctx context.Context) error { return f(ctx) }

// Probe holds named checkers and exposes a net/http health endpoint.
type Probe struct {
	mu       sync.RWMutex
	checkers map[string]Checker
	timeout  time.Duration
}

// New creates a Probe. timeout is the deadline applied to each Handler call.
func New(timeout time.Duration) *Probe {
	return &Probe{
		checkers: make(map[string]Checker),
		timeout:  timeout,
	}
}

// Register adds a named checker. Returns the Probe to allow chaining.
func (p *Probe) Register(name string, c Checker) *Probe {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.checkers[name] = c
	return p
}

type result struct {
	name string
	err  error
}

// Handler returns an http.HandlerFunc that runs all registered checkers concurrently within the request context (bounded by the
// probe timeout). Responds 200 when all pass, 503 when any fail.
func (p *Probe) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), p.timeout)
		defer cancel()

		p.mu.RLock()
		snapshot := make(map[string]Checker, len(p.checkers))
		maps.Copy(snapshot, p.checkers)
		p.mu.RUnlock()

		ch := make(chan result, len(snapshot))
		for name, c := range snapshot {
			go func() {
				ch <- result{name: name, err: c.Check(ctx)}
			}()
		}

		details := make(map[string]string, len(snapshot))
		healthy := true
		for range snapshot {
			res := <-ch
			details[res.name] = "ok"
			if res.err != nil {
				details[res.name] = res.err.Error()
				healthy = false
			}
		}

		code := http.StatusOK
		body := map[string]any{
			"status":  "ok",
			"details": details,
		}
		if !healthy {
			code = http.StatusServiceUnavailable
			body["status"] = "degraded"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(body)
	}
}
