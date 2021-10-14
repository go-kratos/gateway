package tracing

import "io"

type Tracer interface {
	io.Closer

	// Name returns tracer's name
	Name() string
}

type NoopTracer struct {
}

func (n NoopTracer) Closer() error {
	return nil
}

func (n NoopTracer) Name() string {
	return "NoopTracer"
}
