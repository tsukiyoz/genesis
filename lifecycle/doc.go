// Package lifecycle provides irreversible broadcast signals for coordinating
// service startup, readiness, ingress draining, and shutdown.
//
// Each Signal is an independent historical fact. Signaling is thread-safe and
// idempotent, closes a channel to wake current waiters, and remains observable
// by future waiters. Lifecycle groups common service milestones and derives
// liveness, readiness, and ingress-admission decisions from them.
//
// Applications own signal ordering, shutdown delays, runtime cancellation,
// listener draining, and resource cleanup. Reversible dependency health belongs
// in live health probes rather than lifecycle Signals.
package lifecycle
