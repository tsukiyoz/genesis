package lifecycle_test

import (
	"context"
	"fmt"

	"github.com/tsukiyoz/genesis/lifecycle"
)

func ExampleLifecycle() {
	life := lifecycle.New()
	life.RunStarted.Signal()
	life.StartupComplete.Signal()

	fmt.Println(life.ReadinessProbe(context.Background()) == nil)
	fmt.Println(life.AcceptingNewRequests())

	life.ShutdownInitiated.Signal()
	fmt.Println(life.ReadinessProbe(context.Background()) == nil)
	fmt.Println(life.AcceptingNewRequests())

	life.IngressAdmissionClosed.Signal()
	fmt.Println(life.AcceptingNewRequests())
	// Output:
	// true
	// true
	// false
	// true
	// false
}
