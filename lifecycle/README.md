# lifecycle

Package `lifecycle` models service startup and graceful shutdown as independent,
irreversible events. It deliberately does not compress those historical facts
into one current-state enum.

## Signals

A Signal is backed by `sync.Once` and a channel:

```go
type Signal interface {
	Signal()
	Signaled() <-chan struct{}
	Name() string
}
```

Calling `Signal` is thread-safe and idempotent. It closes `Signaled`, waking
every current waiter. Future waiters also observe the already-closed channel
immediately.

```go
event := lifecycle.NewSignal("cache-warmed")

go func() {
	<-event.Signaled()
	startServing()
}()

event.Signal()
```

`IsSignaled` provides a non-blocking historical query.

## Service lifecycle

`Lifecycle` groups the milestones commonly needed by a long-running service:

| Signal | Meaning |
| --- | --- |
| `RunStarted` | The runtime has taken ownership of the service lifetime. |
| `StartupComplete` | Every prerequisite required to serve traffic has started. |
| `HasBeenReady` | At least one complete readiness evaluation has succeeded. |
| `ShutdownInitiated` | Graceful shutdown has begun and readiness must fail. |
| `AfterShutdownDelay` | The load-balancer propagation delay has elapsed or was skipped. |
| `IngressAdmissionClosed` | Business ingress must reject new work. |
| `IngressInFlightDrained` | Accepted ingress work finished or its drain timeout elapsed. |
| `ShutdownComplete` | Workflows and resources have stopped. |

The package does not enforce signal order. The application owns orchestration
and can add application-specific Signals without modifying `Lifecycle`.

## Startup and graceful shutdown

```go
life := lifecycle.New()

life.RunStarted.Signal()
startWorkflows()
waitForStartupPrerequisites()
life.StartupComplete.Signal()

// A shutdown signal arrives.
life.ShutdownInitiated.Signal()
waitDrainDelay()
life.AfterShutdownDelay.Signal()
life.IngressAdmissionClosed.Signal()
shutdownIngress()
life.IngressInFlightDrained.Signal()
runtimeCancel()
waitForWorkflows()
shutdownResources()
life.ShutdownComplete.Signal()
```

Derived behavior is:

| Milestone | Liveness | Readiness | Accepting new requests |
| --- | --- | --- | --- |
| Before `RunStarted` | fail | fail | no |
| After `RunStarted` | pass | fail | no |
| After `StartupComplete` | pass | pass | yes |
| After `ShutdownInitiated` | pass | fail | yes |
| After `IngressAdmissionClosed` | pass | fail | no |
| After `ShutdownComplete` | fail | fail | no |

Readiness and ingress admission are deliberately different switches. During
the load-balancer propagation delay, readiness has already failed while
residual traffic is still accepted.

## Boundaries

- Reversible Redis, NATS, Kafka, or downstream health belongs in separate
  health Probes rather than lifecycle Signals.
- The caller decides when or whether to signal `HasBeenReady`; Genesis does not
  retain readiness results automatically.
- Genesis does not sleep for a shutdown delay, cancel runtime contexts, close
  listeners, wait for in-flight work, or close dependencies.
- An external cancellation should normally initiate drain first. Internal
  runtime contexts should remain alive until ingress and its downstream queues
  have drained.
