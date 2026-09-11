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

type MockTracer struct{}

func (t *MockTracer) Close(context.Context) error { return nil }
func (t *MockTracer) Start(context.Context, string, SpanKind, map[string]string) (
	context.Context, Span) {
	return context.Background(), &MockSpan{}
}

func (t *MockTracer) SpanFromContext(ctx context.Context) (Span, error) {
	return &MockSpan{}, nil
}

func (t *MockTracer) MapCarrier(ctx context.Context) MapCarrier {
	return MapCarrier{}
}

func (t *MockTracer) Extract(ctx context.Context, _ MapCarrier) context.Context {
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

type MockSpan struct{}

func (s *MockSpan) End()                               {}
func (s *MockSpan) SetAttributes(map[string]string)    {}
func (s *MockSpan) IsValid() bool                      { return false }
func (s *MockSpan) IsRecording() bool                  { return false }
func (s *MockSpan) TraceId() string                    { return "" }
func (s *MockSpan) SpanId() string                     { return "" }
func (s *MockSpan) SetStatus(code Code, detail string) {}
