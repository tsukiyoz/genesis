package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestReadyzReport(t *testing.T) {
	handler := New(time.Second, nil, []Check{
		{Name: "scheduler", Probe: func(context.Context) error { return nil }},
		{Name: "nats", Probe: func(context.Context) error { return errors.New("unavailable") }},
	})

	response := httptest.NewRecorder()
	handler.Readyz(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	assertJSONEqual(t, response.Body.Bytes(), `{
		"status":"not_ready",
		"components":{"scheduler":"ok","nats":"failed"}
	}`)
}

func TestReadyzRunsProbesInParallel(t *testing.T) {
	started := make(chan string, 2)
	release := make(chan struct{})
	newProbe := func(name string) Probe {
		return func(ctx context.Context) error {
			started <- name
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	handler := New(time.Second, nil, []Check{
		{Name: "first", Probe: newProbe("first")},
		{Name: "second", Probe: newProbe("second")},
	})

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		response := httptest.NewRecorder()
		handler.Readyz(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		done <- response
	}()

	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			t.Fatal("probes did not start in parallel")
		}
	}
	close(release)

	select {
	case response := <-done:
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
		}
	case <-time.After(time.Second):
		t.Fatal("readiness request did not finish")
	}
}

func TestReadyzTimesOut(t *testing.T) {
	handler := New(20*time.Millisecond, nil, []Check{{
		Name: "slow",
		Probe: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}})

	response := httptest.NewRecorder()
	handler.Readyz(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	assertJSONEqual(t, response.Body.Bytes(), `{
		"status":"not_ready",
		"components":{"slow":"failed"}
	}`)
}

func TestProbePanicIsFailure(t *testing.T) {
	handler := New(time.Second, nil, []Check{{
		Name: "runtime",
		Probe: func(context.Context) error {
			panic("runtime probe failed")
		},
	}})

	response := httptest.NewRecorder()
	handler.Readyz(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func assertJSONEqual(t *testing.T, got []byte, want string) {
	t.Helper()
	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	var wantValue any
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("decode expected JSON: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("response = %s, want %s", got, want)
	}
}
