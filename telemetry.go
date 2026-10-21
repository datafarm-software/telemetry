package telemetry

import (
	"context"
	"fmt"
	"log"

	"github.com/datafarm-software/telemetry/logging"
	"github.com/datafarm-software/telemetry/metering"
	"github.com/datafarm-software/telemetry/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

type Telemetry struct {
	logging.Logger
	metering.Meter
	tracing.Tracer
}

type Opts struct {
	CollectorEndpoint string  `mapstructure:"collectorendpoint" validate:"required"`
	ServiceName       string  `mapstructure:"servicename" validate:"required"`
	TraceRatio        float64 `mapstructure:"traceratio" validate:"gt=0"`
	MeterOpts         []metering.OtlpMeterOption
}

func NewTelemetry(opts Opts) (t *Telemetry, err error) {
	t = &Telemetry{}
	res, err := resource.New(context.Background(), resource.WithContainer(),
		resource.WithAttributes(
			attribute.String("service.name", opts.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("init resource: %v", err)
	}
	t.Logger, err = logging.NewOtlpLogger(
		res, opts.CollectorEndpoint)
	if err != nil {
		return nil, fmt.Errorf("init logger: %v", err)
	}
	t.Tracer, err = tracing.NewOtlpTracer(
		res, opts.ServiceName,
		opts.CollectorEndpoint, opts.TraceRatio)
	if err != nil {
		return nil, fmt.Errorf("init tracer: %v", err)
	}
	t.Meter, err = metering.NewOtlpMeter(metering.OtlpOpts{
		Name:     opts.ServiceName,
		Endpoint: opts.CollectorEndpoint, Res: res,
	}, opts.MeterOpts...)
	if err != nil {
		return nil, fmt.Errorf("init meter: %v", err)
	}
	return t, nil
}

func (t *Telemetry) Close(ctx context.Context) (err error) {
	if err = t.Logger.Close(ctx); err != nil {
		log.Printf("closing logger: %v", err)
	}
	if err = t.Tracer.Close(ctx); err != nil {
		log.Printf("closing tracer: %v", err)
	}
	if err = t.Meter.Close(ctx); err != nil {
		log.Printf("closing meter: %v", err)
	}
	return nil
}

func main() {
	recorder := &metering.OtlpRecorder{}
	_ = metering.Meter(recorder)
	tracer := &tracing.OtlpTracer{}
	_ = tracing.Tracer(tracer)
	logger := &logging.OtlpLogger{}
	_ = logging.Logger(logger)
	_ = logger.LogAccumulator()
}
