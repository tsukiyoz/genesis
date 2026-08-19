package lifecycle

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSignalBroadcastsAndIsIdempotent(t *testing.T) {
	signal := NewSignal("test")
	if signal.Name() != "test" {
		t.Fatalf("signal name = %q, want test", signal.Name())
	}

	const waiters = 64
	var waiting sync.WaitGroup
	var awakened sync.WaitGroup
	var awakenCount atomic.Int32
	waiting.Add(waiters)
	awakened.Add(waiters)
	for range waiters {
		go func() {
			defer awakened.Done()
			waiting.Done()
			<-signal.Signaled()
			awakenCount.Add(1)
		}()
	}
	waiting.Wait()

	var signallers sync.WaitGroup
	signallers.Add(waiters)
	for range waiters {
		go func() {
			defer signallers.Done()
			signal.Signal()
		}()
	}
	signallers.Wait()

	done := make(chan struct{})
	go func() {
		awakened.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("signal did not wake every waiter")
	}
	if got := awakenCount.Load(); got != waiters {
		t.Fatalf("awakened waiters = %d, want %d", got, waiters)
	}
	if !IsSignaled(signal) {
		t.Fatal("signal did not retain its historical state")
	}
	select {
	case <-signal.Signaled():
	default:
		t.Fatal("future waiter did not observe the closed signal")
	}
}

func TestLifecycleSignals(t *testing.T) {
	lifecycle := New()
	ctx := context.Background()

	if !errors.Is(lifecycle.LivenessProbe(ctx), ErrRunNotStarted) {
		t.Fatal("liveness passed before RunStarted")
	}
	if !errors.Is(lifecycle.ReadinessProbe(ctx), ErrStartupIncomplete) {
		t.Fatal("readiness passed before StartupComplete")
	}
	if lifecycle.AcceptingNewRequests() {
		t.Fatal("admission opened before StartupComplete")
	}

	lifecycle.RunStarted.Signal()
	if err := lifecycle.LivenessProbe(ctx); err != nil {
		t.Fatalf("liveness after RunStarted: %v", err)
	}

	lifecycle.StartupComplete.Signal()
	if err := lifecycle.ReadinessProbe(ctx); err != nil {
		t.Fatalf("readiness after StartupComplete: %v", err)
	}
	if !lifecycle.AcceptingNewRequests() {
		t.Fatal("admission remained closed after StartupComplete")
	}

	lifecycle.HasBeenReady.Signal()
	lifecycle.ShutdownInitiated.Signal()
	if !errors.Is(lifecycle.ReadinessProbe(ctx), ErrShutdownInProgress) {
		t.Fatal("readiness passed after ShutdownInitiated")
	}
	if !lifecycle.AcceptingNewRequests() {
		t.Fatal("admission closed before the load-balancer delay ended")
	}

	lifecycle.AfterShutdownDelay.Signal()
	if !lifecycle.AcceptingNewRequests() {
		t.Fatal("AfterShutdownDelay implicitly closed admission")
	}
	lifecycle.IngressAdmissionClosed.Signal()
	if lifecycle.AcceptingNewRequests() {
		t.Fatal("admission remained open after IngressAdmissionClosed")
	}
	if err := lifecycle.LivenessProbe(ctx); err != nil {
		t.Fatalf("liveness failed before ShutdownComplete: %v", err)
	}

	lifecycle.IngressInFlightDrained.Signal()
	lifecycle.ShutdownComplete.Signal()
	if !errors.Is(lifecycle.LivenessProbe(ctx), ErrShutdownComplete) {
		t.Fatal("liveness passed after ShutdownComplete")
	}
	for _, signal := range []Signal{
		lifecycle.RunStarted,
		lifecycle.StartupComplete,
		lifecycle.HasBeenReady,
		lifecycle.ShutdownInitiated,
		lifecycle.AfterShutdownDelay,
		lifecycle.IngressAdmissionClosed,
		lifecycle.IngressInFlightDrained,
		lifecycle.ShutdownComplete,
	} {
		if !IsSignaled(signal) {
			t.Fatalf("historical event %s was lost", signal.Name())
		}
	}
}

func TestLifecycleProbesHonorContext(t *testing.T) {
	lifecycle := New()
	lifecycle.RunStarted.Signal()
	lifecycle.StartupComplete.Signal()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(lifecycle.LivenessProbe(ctx), context.Canceled) {
		t.Fatal("liveness ignored canceled context")
	}
	if !errors.Is(lifecycle.ReadinessProbe(ctx), context.Canceled) {
		t.Fatal("readiness ignored canceled context")
	}
}

func TestIsSignaledNil(t *testing.T) {
	if IsSignaled(nil) {
		t.Fatal("nil signal reported signaled")
	}
}
