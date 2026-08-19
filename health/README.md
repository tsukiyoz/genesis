# health

Package `health` exposes framework-independent `net/http` liveness and
readiness handlers backed by named probes.

## Usage

```go
liveChecks := []health.Check{
	{Name: "lifecycle", Probe: life.LivenessProbe},
}
readyChecks := []health.Check{
	{Name: "lifecycle", Probe: life.ReadinessProbe},
	{Name: "redis", Probe: redisProbe},
}

handler := health.New(time.Second, liveChecks, readyChecks)

mux := http.NewServeMux()
mux.HandleFunc("GET /livez", handler.Livez)
mux.HandleFunc("GET /readyz", handler.Readyz)
```

`Handler` does not create a listener or own an `http.Server`. The application
registers the handlers on its existing router and owns server shutdown.

## Probe execution

Every check in one report runs concurrently beneath both the HTTP request
context and one overall timeout. A slow component therefore does not delay
other checks serially.

```go
type Probe func(context.Context) error
```

A Probe must honor its context. Returning `nil` reports `ok`; returning an
error or panicking reports `failed`. Probe errors and panic values are not
included in the HTTP response, avoiding accidental disclosure of dependency
details.

If the configured timeout is not positive, the handler uses a one-second
default. Check names within one liveness or readiness set must be stable and
unique because the response is keyed by component name.

## Responses

A successful readiness response is:

```http
HTTP/1.1 200 OK
Cache-Control: no-store
Content-Type: application/json

{"status":"ready","components":{"lifecycle":"ok","redis":"ok"}}
```

If any readiness Probe fails or times out, the endpoint returns `503`:

```json
{
  "status": "not_ready",
  "components": {
    "lifecycle": "ok",
    "redis": "failed"
  }
}
```

Liveness uses `alive` and `not_alive` as the top-level status values. Readiness
uses `ready` and `not_ready`.

## Concurrency contract

The result channel is buffered to the number of checks, so a Handler that has
already returned after its timeout does not leave a context-aware Probe blocked
while reporting its result. A Probe that ignores context and blocks forever can
still leak its own goroutine; the package cannot forcibly terminate arbitrary
Go code.
