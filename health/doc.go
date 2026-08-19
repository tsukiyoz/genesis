// Package health provides framework-independent HTTP liveness and readiness
// handlers backed by named probes.
//
// All probes in one report run concurrently beneath the request context and a
// single overall timeout. Probe implementations must honor their context.
// Returning nil reports a healthy component; returning an error or panicking
// reports a failed component.
//
// Handler owns neither a listener nor an http.Server. Applications register
// Livez and Readyz on their existing HTTP router and retain transport lifecycle
// ownership.
package health
