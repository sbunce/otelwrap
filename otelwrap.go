/*
Package trace is used for tracing.

	Background:
	This package is a wrapper for the otel package for doing tracing.
*/
package otelwrap

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// provider used for all traces.
var provider = otel.GetTracerProvider().Tracer("")

// noopSpan does nothing.
var noopSpan = &Span{
	noop: true,
}

// Span of a trace.
type Span struct {
	// noop is a span that is a no-op.
	noop bool

	// Only set if noop=false.
	span trace.Span
}

// SpanFromContext returns a span from the context. If there is no span in the
// context in the no-op span is returned.
func SpanFromContext(ctx context.Context) *Span {
	return newSpan(trace.SpanFromContext(ctx))
}

// newSpan returns a new function span based on otel span.
func newSpan(span trace.Span) *Span {
	return &Span{
		span: span,
	}
}

// AddEvent records an event.
func (s *Span) AddEvent(name string) {
	if s.noop {
		return
	}
	s.span.AddEvent(name)
}

// End the span.
func (s *Span) End() {
	if s.noop {
		return
	}
	s.span.End()
}

// RecordError records an error in the span.
func (s *Span) RecordError(err error) {
	if s.noop {
		return
	}
	s.span.RecordError(err)
}

// StartInternal trace within a process. Params show in the span name.
func StartInternal(ctx context.Context, params ...string) (context.Context, *Span) {
	return start(ctx, trace.SpanKindInternal, params...)
}

// StartClient trace when calling another process. Params show in the span name.
func StartClient(ctx context.Context, params ...string) (context.Context, *Span) {
	return start(ctx, trace.SpanKindClient, params...)
}

// StartConsumer trace when consuming from a pub/sub system. Params show in the
// span name.
func StartConsumer(ctx context.Context, params ...string) (context.Context, *Span) {
	return start(ctx, trace.SpanKindConsumer, params...)
}

// StartProducer trace when sending to a pub/sub system. Params show in the span
// name.
func StartProducer(ctx context.Context, params ...string) (context.Context, *Span) {
	return start(ctx, trace.SpanKindProducer, params...)
}

// StartServer trace when another process is calling us. Params show in the span
// name.
func StartServer(ctx context.Context, params ...string) (context.Context, *Span) {
	return start(ctx, trace.SpanKindServer, params...)
}

func start(ctx context.Context, kind trace.SpanKind, params ...string) (context.Context, *Span) {
	info := caller(3)
	var parts []string
	for _, param := range params {
		parts = append(parts, fmt.Sprintf("%q", param))
	}
	name := fmt.Sprintf("%s(%s)", info.Func, strings.Join(parts, ", "))
	ctx, span := provider.Start(ctx, name, trace.WithSpanKind(kind))
	span.SetAttributes(attribute.KeyValue{
		Key:   "file",
		Value: attribute.StringValue(fmt.Sprintf("%s:%d", info.File, info.Line)),
	})
	return ctx, newSpan(span)
}

type callerInfo struct {
	File string
	Line int
	Func string
}

func caller(n int) callerInfo {
	pc, file, line, ok := runtime.Caller(n)
	if !ok {
		return callerInfo{
			File: "unknown",
			Func: "unknown",
		}
	}
	fn := runtime.FuncForPC(pc)
	parts := strings.Split(fn.Name(), "/")
	return callerInfo{
		File: file,
		Line: line,
		Func: parts[len(parts)-1],
	}
}

// Export is used to send a trace ID to another process.
func Export(ctx context.Context) map[string]string {
	index := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, carrier(index))
	return index
}

// Import is used to receive a trace ID from another process.
func Import(ctx context.Context, index map[string]string) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, carrier(index))
}

// carrier conforms to propagation.TextMapCarrier.
type carrier map[string]string

func (m carrier) Get(key string) string {
	return m[key]
}

func (m carrier) Set(key string, value string) {
	m[key] = value
}

func (m carrier) Keys() []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

