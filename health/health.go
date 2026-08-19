package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const defaultTimeout = time.Second

const (
	componentOK     = "ok"
	componentFailed = "failed"
)

// Probe reports whether one component is healthy. Implementations must honor
// ctx so a timed-out HTTP request does not leave a probe goroutine behind.
type Probe func(ctx context.Context) error

// Check associates a stable component name with its probe.
// Names within one probe set should be unique.
type Check struct {
	Name  string
	Probe Probe
}

// Handler serves liveness and readiness reports. Probes in a report run in
// parallel under one overall timeout.
type Handler struct {
	timeout     time.Duration
	liveChecks  []Check
	readyChecks []Check
}

// New constructs a Handler. Slices are copied so callers may safely reuse
// their input after construction.
func New(timeout time.Duration, liveChecks, readyChecks []Check) *Handler {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Handler{
		timeout:     timeout,
		liveChecks:  append([]Check(nil), liveChecks...),
		readyChecks: append([]Check(nil), readyChecks...),
	}
}

// Livez writes the process liveness report.
func (h *Handler) Livez(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, "alive", "not_alive", h.liveChecks)
}

// Readyz writes the traffic readiness report.
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, "ready", "not_ready", h.readyChecks)
}

type report struct {
	Status     string            `json:"status"`
	Components map[string]string `json:"components"`
}

type probeResult struct {
	index  int
	status string
}

func (h *Handler) serve(
	w http.ResponseWriter,
	r *http.Request,
	success, failure string,
	checks []Check,
) {
	components, healthy := h.run(r.Context(), checks)
	status := success
	statusCode := http.StatusOK
	if !healthy {
		status = failure
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(report{Status: status, Components: components})
}

func (h *Handler) run(parent context.Context, checks []Check) (map[string]string, bool) {
	ctx, cancel := context.WithTimeout(parent, h.timeout)
	defer cancel()

	results := make(chan probeResult, len(checks))
	for i, check := range checks {
		go runProbe(ctx, i, check, results)
	}

	statuses := make([]string, len(checks))
	healthy := true
	for received := 0; received < len(checks); received++ {
		select {
		case result := <-results:
			statuses[result.index] = result.status
			if result.status != componentOK {
				healthy = false
			}
		case <-ctx.Done():
			healthy = false
			for i := range statuses {
				if statuses[i] == "" {
					statuses[i] = componentFailed
				}
			}
			return makeComponents(checks, statuses), healthy
		}
	}

	return makeComponents(checks, statuses), healthy
}

func runProbe(ctx context.Context, index int, check Check, results chan<- probeResult) {
	result := probeResult{index: index, status: componentFailed}
	defer func() {
		_ = recover()
		results <- result
	}()

	if check.Probe != nil && check.Probe(ctx) == nil {
		result.status = componentOK
	}
}

func makeComponents(checks []Check, statuses []string) map[string]string {
	components := make(map[string]string, len(checks))
	for i, check := range checks {
		components[check.Name] = statuses[i]
	}
	return components
}
