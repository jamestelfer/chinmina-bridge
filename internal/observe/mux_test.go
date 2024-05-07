package observe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"

	"github.com/stretchr/testify/assert"
)

func TestHandleRouteTag(t *testing.T) {
	rawMux := http.NewServeMux()
	mux := NewMux(rawMux)

	var routeLabels []attribute.KeyValue

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// the otel handler middleware adds the labeler, so this is also
		// indirectly testing the presence of that middleware configuration in
		// the observe muxer
		labels, _ := otelhttp.LabelerFromContext(r.Context())

		routeLabels = labels.Get()
	})
	mux.Handle("/test", testHandler)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(context.Background())

	mux.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code, "Expected HTTP status OK")
	assert.Equal(t, []attribute.KeyValue{attribute.String("http.route", "/test")}, routeLabels)
}
