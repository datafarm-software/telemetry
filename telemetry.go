package telemetry

import (
	"github.com/datafarm-software/telemetry/logging"
	"github.com/datafarm-software/telemetry/metering"
	"github.com/datafarm-software/telemetry/tracing"
)

type Opts struct {
	CollectorEndpoint string `mapstructure:"collectorendpoint" validate:"required"`
}

func main() {
	recorder := &metering.OtlpRecorder{}
	_ = metering.Meter(recorder)
	tracer := &tracing.OtlpTracer{}
	_ = tracing.Tracer(tracer)
	logger := &logging.OtlpLogger{}
	_ = logging.Logger(logger)
}
