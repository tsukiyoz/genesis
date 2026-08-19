# Genesis

Genesis provides small, composable runtime primitives for Go services. It is
framework-independent and depends only on the Go standard library.

## Packages

| Package | Purpose |
| --- | --- |
| [`health`](./health) | Concurrent HTTP liveness and readiness probes under one overall timeout. |
| [`lifecycle`](./lifecycle) | Irreversible broadcast lifecycle signals with derived health and ingress-admission decisions. |

Each package has its own README, package documentation, and executable examples.

## Installation

Genesis is one Go module with one version shared by every package:

```bash
go get github.com/tsukiyoz/genesis@latest
```

Applications import only the packages they use:

```go
import (
	"github.com/tsukiyoz/genesis/health"
	"github.com/tsukiyoz/genesis/lifecycle"
)
```

## Design principles

- Provide primitives rather than an application framework.
- Keep HTTP handlers independent of Echo, Gin, gRPC, and other transports.
- Keep lifecycle observation separate from runtime cancellation and shutdown delays.
- Treat reversible dependency health as a live Probe, not a historical lifecycle event.
- Make goroutine ownership, cancellation, and shutdown ordering explicit in the application.
- Add a new package only after its semantics have been validated by real services.

Genesis does not own HTTP listeners, OS signal handling, application
configuration, dependency-specific probes, or resource shutdown.

## Documentation

- [`health` package guide](./health/README.md)
- [`lifecycle` package guide](./lifecycle/README.md)
- [`health` on pkg.go.dev](https://pkg.go.dev/github.com/tsukiyoz/genesis/health)
- [`lifecycle` on pkg.go.dev](https://pkg.go.dev/github.com/tsukiyoz/genesis/lifecycle)

Genesis is currently pre-v1. Until `v1.0.0`, minor releases may refine public
APIs as the packages are adopted by additional services.
