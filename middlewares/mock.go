package middlewares

import (
	"context"
	"fmt"

	"github.com/datafarm-software/telemetry"
)

type mockProvider struct {
	*telemetry.Telemetry
}

func NewMockProvider(t *telemetry.Telemetry) (
	MiddlewareProvider[context.Context], error,
) {
	if t == nil {
		return nil, fmt.Errorf("received nil telemetry")
	}
	return &mockProvider{
		Telemetry: t,
	}, nil
}

func (m *mockProvider) TelemetryMiddlewares() Middlewares[context.Context] {
	return []func(ctx context.Context, next func(context.Context)){
		m.CountApiRequest, m.TraceRequest, m.LogRequest, m.RecordLatency,
	}
}

func (m *mockProvider) RecordLatency(ctx context.Context, next func(context.Context)) {
	next(ctx)
}

func (m *mockProvider) CountApiRequest(ctx context.Context, next func(context.Context)) {
	next(ctx)
}

func (m *mockProvider) TraceRequest(ctx context.Context, next func(context.Context)) {
	next(ctx)
}

func (m *mockProvider) LogRequest(ctx context.Context, next func(context.Context)) {
	log := m.Logger.LogAccumulator()
	ctx = context.WithValue(ctx, "request-log", log)
	next(ctx)
}
