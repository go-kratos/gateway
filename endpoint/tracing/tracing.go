package tracing

import (
	"io"
)

// Name of the middleware tracing
const Otel = "opentelemetery"

type Tracer interface {
	io.Closer

	// Name returns tracer's name
	Name() string
}

type NoopTracer struct {
}

func (n NoopTracer) Close() error {
	return nil
}

func (n NoopTracer) Name() string {
	return "NoopTracer"
}
