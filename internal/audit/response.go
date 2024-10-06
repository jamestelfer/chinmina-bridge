package audit

import (
	"bufio"
	"net"
	"net/http"
)

func wrapResponseWriter(w http.ResponseWriter, e *Entry) http.ResponseWriter {
	wrapped := &responseWrapper{responseWriter: w, entry: e}
	if _, ok := w.(http.Hijacker); ok {
		return &hijackWrapper{*wrapped}
	}
	return wrapped
}

type responseWrapper struct {
	responseWriter http.ResponseWriter
	entry          *Entry
}

func (w *responseWrapper) Header() http.Header {
	return w.responseWriter.Header()
}

func (w *responseWrapper) Write(buf []byte) (int, error) {
	return w.responseWriter.Write(buf)
}

func (w *responseWrapper) WriteHeader(code int) {
	w.entry.Status = code
	w.responseWriter.WriteHeader(code)
}

func (w *responseWrapper) Flush() {
	if flusher, ok := w.responseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// hijackWrapper wraps a response writer that supports hijacking.
type hijackWrapper struct {
	responseWrapper
}

func (h *hijackWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return h.responseWriter.(http.Hijacker).Hijack()
}
