package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type OtlpTracer struct {
	trace.Tracer
	*sdktrace.TracerProvider
}

func NewOtlpTracer(res *resource.Resource, name, endpoint string) (
	Tracer, error) {
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("init exporter: %v", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	tracer := &OtlpTracer{
		Tracer:         tp.Tracer(name),
		TracerProvider: tp,
	}
	return tracer, nil
}

func (o *OtlpTracer) Close(ctx context.Context) error {
	return o.TracerProvider.Shutdown(ctx)
}

func (o *OtlpTracer) Start(ctx context.Context, name string, kind SpanKind,
	mapAttrs map[string]string) (retCtx context.Context, s Span) {
	attrs := make([]attribute.KeyValue, 0, len(mapAttrs)+1)
	for k, v := range mapAttrs {
		if k == "" {
			continue
		}
		attrs = append(attrs, attribute.String(k, v))
	}
	os := &OtlpSpan{}
	retCtx, os.Span = o.Tracer.Start(ctx, name,
		trace.WithSpanKind(trace.SpanKind(kind)),
		trace.WithAttributes(attrs...))
	return retCtx, os
}

func (o *OtlpTracer) MapCarrier(ctx context.Context) MapCarrier {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return MapCarrier(carrier)
}

func (o *OtlpTracer) Extract(ctx context.Context, carrier MapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(
		ctx, propagation.MapCarrier(carrier),
	)
}

func (o *OtlpTracer) SpanFromContext(ctx context.Context) (Span, error) {
	span := trace.SpanFromContext(ctx)
	var err error
	if !span.SpanContext().IsValid() {
		err = SpanNotFound
	}
	//NOTE: always able to return span because it is a no op if not found
	return &OtlpSpan{span}, err
}

type OtlpSpan struct {
	trace.Span
}

func (o *OtlpSpan) End() {
	o.Span.End()
}
func (o *OtlpSpan) SetAttributes(mapAttrs map[string]string) {
	attrs := make([]attribute.KeyValue, 0, len(mapAttrs))
	for k, v := range mapAttrs {
		if k == "" {
			continue
		}
		attrs = append(attrs, attribute.String(k, v))
	}
	o.Span.SetAttributes(attrs...)
}

func (o *OtlpSpan) IsValid() bool {
	return o.SpanContext().IsValid()
}

func (o *OtlpSpan) IsRecording() bool {
	return o.IsRecording()
}

func (o *OtlpSpan) TraceId() string {
	return o.Span.SpanContext().TraceID().String()
}

func (o *OtlpSpan) SpanId() string {
	return o.Span.SpanContext().SpanID().String()
}

func (o *OtlpSpan) SetStatus(code Code, detail string) {
	o.Span.SetStatus(codes.Code(code), detail)
}
