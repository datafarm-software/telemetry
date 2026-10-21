package middlewares

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/datafarm-software/telemetry"
	"github.com/datafarm-software/telemetry/logging"
	"github.com/datafarm-software/telemetry/tracing"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type humaProvider struct {
	*telemetry.Telemetry
}

func NewHumaProvider(t *telemetry.Telemetry) (
	mp MiddlewareProvider[huma.Context], err error,
) {
	if t == nil {
		return nil, fmt.Errorf("received nil telemetry")
	}
	mp = &humaProvider{Telemetry: t}
	return
}

func (m *humaProvider) TelemetryMiddlewares() Middlewares[huma.Context] {
	return []func(ctx huma.Context, next func(huma.Context)){
		m.CountApiRequest, m.TraceRequest, m.LogRequest, m.RecordLatency,
	}
}

func (m *humaProvider) RecordLatency(humaCtx huma.Context, next func(huma.Context)) {
	r, w := humamux.Unwrap(humaCtx)
	ctx := r.Context()
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if err := m.Telemetry.Meter.RecordLatency(ctx, duration); err != nil {
			w.Write([]byte("Internal error while recording latency of the request. Please notify a system administrator."))
			log.Printf("recordLatency: %v", err)
		}
	}()
	next(humaCtx)
}

func (m *humaProvider) CountApiRequest(humaCtx huma.Context, next func(huma.Context)) {
	path := getPath(humaCtx)
	m.Telemetry.Meter.CountApiRequest(humaCtx.Context(), 1,
		map[string]string{
			"http.route":  path,
			"http.method": humaCtx.Method(),
		},
	)
	next(humaCtx)
}

func (m *humaProvider) TraceRequest(humaCtx huma.Context, next func(huma.Context)) {
	path := getPath(humaCtx)
	headerMap := make(map[string]string)
	humaCtx.EachHeader(func(name, value string) {
		headerMap[name] = value
	})
	ctx := otel.GetTextMapPropagator().Extract(
		humaCtx.Context(), propagation.MapCarrier(headerMap),
	)
	var span tracing.Span
	ctx, span = m.Telemetry.Tracer.Start(
		ctx, path, tracing.SpanKind(trace.SpanKindServer),
		map[string]string{"http.method": humaCtx.Method()},
	)
	defer span.End()
	r, w := humamux.Unwrap(humaCtx)
	r.WithContext(ctx)
	next(humamux.NewContext(humaCtx.Operation(), r, w))
}

func (m *humaProvider) LogRequest(humaCtx huma.Context, next func(huma.Context)) {
	log := m.Telemetry.Logger.LogAccumulator()
	log.AddMetadata(logging.Metadata{
		"http.method": {humaCtx.Method()},
		"http.route":  {getPath(humaCtx)},
	})
	span, _ := m.Telemetry.Tracer.SpanFromContext(humaCtx.Context())
	if span.IsValid() {
		log.AddMetadata(logging.Metadata{
			"trace_id": {span.TraceId()},
			"span_id":  {span.SpanId()},
		})
	}
	humaCtx = huma.WithValue(humaCtx, "request-log", log)
	next(humaCtx)
	log.AddMetadata(logging.Metadata{
		"http.status_code": {fmt.Sprint(humaCtx.Status())},
	})
	switch getFirstDigit(humaCtx.Status()) {
	case 4:
		m.Telemetry.Logger.Warn("HTTP Client Error", log.Metadata())
	case 5:
		m.Telemetry.Logger.Error("HTTP Internal Error", log.Metadata())
	default:
		m.Telemetry.Logger.Info("HTTP Client Request", log.Metadata())
	}
}

func getFirstDigit(n int) int {
	n = int(math.Abs(float64(n)))
	for n >= 10 {
		n /= 10
	}
	return n
}

func getPath(humaCtx huma.Context) (path string) {
	r, _ := humamux.Unwrap(humaCtx)
	path = r.URL.Path
	route := mux.CurrentRoute(r)
	if route != nil {
		path, _ = route.GetPathTemplate()
	}
	return
}
