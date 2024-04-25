package observe

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Multiplexer interface {
	Handle(pattern string, handler http.Handler)
	http.Handler
}

type Mux struct {
	wrapped Multiplexer
	handler http.Handler
}

func NewMux(wrapped Multiplexer) *Mux {
	return &Mux{
		wrapped: wrapped,
		handler: otelhttp.NewHandler(wrapped, "/"),
	}
}

func (mux *Mux) Handle(pattern string, handler http.Handler) {
	// Configure the "http.route" for the HTTP instrumentation.
	taggedHandler := otelhttp.WithRouteTag(pattern, handler)
	mux.wrapped.Handle(pattern, taggedHandler)
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.handler.ServeHTTP(w, r)
}
