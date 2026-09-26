package tracing

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var SpanNotFound = errors.New("no span in context")

type SpanKind trace.SpanKind

const (
	SpanKindUnspecified SpanKind = 0
	SpanKindInternal    SpanKind = 1
	SpanKindServer      SpanKind = 2
	SpanKindClient      SpanKind = 3
	SpanKindProducer    SpanKind = 4
	SpanKindConsumer    SpanKind = 5
)

type MapCarrier propagation.MapCarrier
type Code codes.Code

type Tracer interface {
	Close(context.Context) error
	Start(context.Context, string, SpanKind, map[string]string) (context.Context, Span)
	//NOTE: could return SpanNotFound
	SpanFromContext(context.Context) (Span, error)
	MapCarrier(context.Context) MapCarrier
	Extract(context.Context, MapCarrier) context.Context
}

func MockTracer() Tracer { return &mockTracer{} }

type mockTracer struct{}

func (t *mockTracer) Close(context.Context) error { return nil }
func (t *mockTracer) Start(context.Context, string, SpanKind, map[string]string) (
	context.Context, Span) {
	return context.Background(), &mockSpan{}
}

func (t *mockTracer) SpanFromContext(ctx context.Context) (Span, error) {
	return &mockSpan{}, nil
}

func (t *mockTracer) MapCarrier(ctx context.Context) MapCarrier {
	return MapCarrier{}
}

func (t *mockTracer) Extract(ctx context.Context, _ MapCarrier) context.Context {
	return ctx
}

type Span interface {
	End()
	SetAttributes(map[string]string)
	IsValid() bool
	IsRecording() bool
	TraceId() string
	SpanId() string
	SetStatus(code Code, detail string)
}

func MockSpan() Span { return &mockSpan{} }

type mockSpan struct{}

func (s *mockSpan) End()                               {}
func (s *mockSpan) SetAttributes(map[string]string)    {}
func (s *mockSpan) IsValid() bool                      { return false }
func (s *mockSpan) IsRecording() bool                  { return false }
func (s *mockSpan) TraceId() string                    { return "" }
func (s *mockSpan) SpanId() string                     { return "" }
func (s *mockSpan) SetStatus(code Code, detail string) {}
