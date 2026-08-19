package lifecycle

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrRunNotStarted means the service runtime has not taken ownership of its lifecycle.
	ErrRunNotStarted = errors.New("service run has not started")
	// ErrStartupIncomplete means the service has not completed its startup prerequisites.
	ErrStartupIncomplete = errors.New("service startup is incomplete")
	// ErrShutdownInProgress means graceful shutdown has begun.
	ErrShutdownInProgress = errors.New("service shutdown is in progress")
	// ErrShutdownComplete means the service has finished shutting down.
	ErrShutdownComplete = errors.New("service shutdown is complete")
)

// Signal represents one irreversible lifecycle event. Signal is idempotent;
// closing Signaled broadcasts to every current and future waiter.
type Signal interface {
	Signal()
	Signaled() <-chan struct{}
	Name() string
}

// NewSignal creates an unsignaled named lifecycle event.
func NewSignal(name string) Signal {
	return &signal{name: name, signaled: make(chan struct{})}
}

type signal struct {
	name     string
	once     sync.Once
	signaled chan struct{}
}

func (s *signal) Signal() {
	s.once.Do(func() {
		close(s.signaled)
	})
}

func (s *signal) Signaled() <-chan struct{} {
	return s.signaled
}

func (s *signal) Name() string {
	return s.name
}

// IsSignaled reports whether an irreversible event has occurred.
func IsSignaled(signal Signal) bool {
	if signal == nil {
		return false
	}
	select {
	case <-signal.Signaled():
		return true
	default:
		return false
	}
}

// Lifecycle groups common service lifecycle events. The events intentionally
// remain independent historical facts instead of being compressed into one
// current-state enum. Applications own their triggering order.
type Lifecycle struct {
	// RunStarted is signaled when Run takes ownership of the service lifetime.
	RunStarted Signal
	// StartupComplete is signaled after every prerequisite needed to serve traffic has started.
	StartupComplete Signal
	// HasBeenReady records that one complete readiness evaluation has succeeded at least once.
	HasBeenReady Signal
	// ShutdownInitiated makes readiness fail before ingress or dependencies stop.
	ShutdownInitiated Signal
	// AfterShutdownDelay records the end of the load-balancer propagation window.
	AfterShutdownDelay Signal
	// IngressAdmissionClosed makes every business ingress reject new work.
	IngressAdmissionClosed Signal
	// IngressInFlightDrained records that accepted ingress work finished or timed out.
	IngressInFlightDrained Signal
	// ShutdownComplete records that workflows and resources have stopped.
	ShutdownComplete Signal
}

// New creates a lifecycle whose events are all initially unsignaled.
func New() *Lifecycle {
	return &Lifecycle{
		RunStarted:             NewSignal("RunStarted"),
		StartupComplete:        NewSignal("StartupComplete"),
		HasBeenReady:           NewSignal("HasBeenReady"),
		ShutdownInitiated:      NewSignal("ShutdownInitiated"),
		AfterShutdownDelay:     NewSignal("AfterShutdownDelay"),
		IngressAdmissionClosed: NewSignal("IngressAdmissionClosed"),
		IngressInFlightDrained: NewSignal("IngressInFlightDrained"),
		ShutdownComplete:       NewSignal("ShutdownComplete"),
	}
}

// LivenessProbe succeeds after RunStarted and until ShutdownComplete.
func (l *Lifecycle) LivenessProbe(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !IsSignaled(l.RunStarted) {
		return ErrRunNotStarted
	}
	if IsSignaled(l.ShutdownComplete) {
		return ErrShutdownComplete
	}
	return nil
}

// ReadinessProbe succeeds after StartupComplete and before ShutdownInitiated.
// Reversible dependency health belongs in separate readiness probes.
func (l *Lifecycle) ReadinessProbe(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !IsSignaled(l.StartupComplete) {
		return ErrStartupIncomplete
	}
	if IsSignaled(l.ShutdownInitiated) {
		return ErrShutdownInProgress
	}
	return nil
}

// AcceptingNewRequests reports whether startup completed and ingress admission
// has not closed. It intentionally remains true during the shutdown delay,
// after readiness has already failed.
func (l *Lifecycle) AcceptingNewRequests() bool {
	return IsSignaled(l.StartupComplete) && !IsSignaled(l.IngressAdmissionClosed)
}
