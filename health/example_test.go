package health_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/tsukiyoz/genesis/health"
)

func ExampleHandler() {
	handler := health.New(time.Second, nil, []health.Check{{
		Name:  "lifecycle",
		Probe: func(context.Context) error { return nil },
	}})

	response := httptest.NewRecorder()
	handler.Readyz(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	fmt.Println(response.Code)
	fmt.Print(response.Body.String())
	// Output:
	// 200
	// {"status":"ready","components":{"lifecycle":"ok"}}
}
