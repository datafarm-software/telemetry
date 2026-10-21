package middlewares

type Middlewares[C any] []func(ctx C, next func(C))

type MiddlewareProvider[C any] interface {
	TelemetryMiddlewares() Middlewares[C]
	RecordLatency(ctx C, next func(C))
	CountApiRequest(ctx C, next func(C))
	TraceRequest(ctx C, next func(C))
	LogRequest(ctx C, next func(C))
}
