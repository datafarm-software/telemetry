package metering

import (
	"context"
	"fmt"
	"log"
	"maps"
	"runtime"
	"sync/atomic"
	"time"

	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

type OtlpMeterOption func(*OtlpRecorder) error

type OtlpRecorder struct {
	requestLatency                             metric.Float64Histogram
	apiCounter                                 metric.Int64Counter
	memoryGauge, uptimeGauge, activeUsersGauge metric.Int64ObservableGauge
	name                                       string
	mp                                         *sdkmetric.MeterProvider
	activeUsersCount                           atomic.Int64
	otelRunTimeStarted                         bool
}

type OtlpOpts struct {
	Name, Endpoint string
	Res            *resource.Resource
}

// NOTE: This uses insecure HTTP
func NewOtlpMeter(otlpOpts OtlpOpts, meterOpts ...OtlpMeterOption) (
	Meter, error) {
	exporter, err := otlpmetrichttp.New(
		context.Background(), otlpmetrichttp.WithEndpoint(otlpOpts.Endpoint),
		otlpmetrichttp.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("init exporter: %v", err)
	}
	reader := sdkmetric.NewPeriodicReader(exporter,
		sdkmetric.WithProducer(otelruntime.NewProducer()))
	sdkOpts := []sdkmetric.Option{sdkmetric.WithReader(reader)}
	if otlpOpts.Res != nil {
		sdkOpts = append(sdkOpts, sdkmetric.WithResource(otlpOpts.Res))
	}
	mp := sdkmetric.NewMeterProvider(sdkOpts...)
	o := &OtlpRecorder{name: otlpOpts.Name, mp: mp}
	for _, opt := range meterOpts {
		if err = opt(o); err != nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("meter option: %v", err)
	}
	mp := sdkmetric.NewMeterProvider(sdkOpts...)
	o := &OtlpRecorder{mp: mp}
	return o, nil
}

func (o *OtlpRecorder) Close(ctx context.Context) error {
	return o.mp.Shutdown(ctx)
}

func WithOtelRuntime() OtlpMeterOption {
	return func(o *OtlpRecorder) error {
		return o.OtelRunTime(o.name)
	}
}

func (o *OtlpRecorder) OtelRunTime(_ string) (err error) {
	if o.otelRunTimeStarted {
		return
	}
	if err = otelruntime.Start(); err != nil {
		return fmt.Errorf("start: %v", err)
	}
	o.otelRunTimeStarted = true
	return
}

var processStart = time.Now().Unix()

func WithUptime() OtlpMeterOption {
	return func(o *OtlpRecorder) error {
		return o.Uptime(o.name)
	}
}

func (o *OtlpRecorder) Uptime(name string) (err error) {
	if o.uptimeGauge != nil {
		return
	}
	meter := o.mp.Meter(name)
	o.uptimeGauge, err = meter.Int64ObservableGauge(
		name+".uptime",
		metric.WithDescription("Process Uptime"),
		metric.WithUnit("s"),
		metric.WithInt64Callback(func(_ context.Context, ob metric.Int64Observer) error {
			ob.Observe(processStart)
			return nil
		}),
	)
	if err != nil {
		err = fmt.Errorf("init uptime gauge: %v", err)
	}
	return err
}

func WithMemoryUsage() OtlpMeterOption {
	return func(o *OtlpRecorder) error {
		return o.MemoryUsage(o.name)
	}
}

func (o *OtlpRecorder) MemoryUsage(name string) (err error) {
	if o.memoryGauge != nil {
		return
	}
	meter := o.mp.Meter(name)
	o.memoryGauge, err = meter.Int64ObservableGauge(
		name+".memory.heap",
		metric.WithDescription("Memory usage of the allocated heap objects."),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			o.Observe(int64(m.HeapAlloc))
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("init memory gauge: %v", err)
	}
	return err
}

func WithLatency() OtlpMeterOption {
	return func(o *OtlpRecorder) error {
		return o.setupRequestLatency(o.name)
	}
}

func (o *OtlpRecorder) setupRequestLatency(name string) (err error) {
	meter := o.mp.Meter(name)
	o.requestLatency, err = meter.Float64Histogram(
		name+".task.duration",
		metric.WithDescription("The duration of task execution."),
		metric.WithUnit("s"),
	)
	if err != nil {
		err = fmt.Errorf("init histogram: %v", err)
	}
	return err
}

func (o *OtlpRecorder) RecordLatency(ctx context.Context, dur time.Duration) (err error) {
	if o.requestLatency == nil {
		if err = o.setupRequestLatency(o.name); err != nil {
			return fmt.Errorf("setupRequestLatency: %v", err)
		}
	}
	o.requestLatency.Record(ctx, float64(dur))
	return nil
}

func WithActiveUsersCount() OtlpMeterOption {
	return func(o *OtlpRecorder) error {
		return o.setupActiveUsersGauge(o.name)
	}
}

func (o *OtlpRecorder) setupActiveUsersGauge(name string) (err error) {
	meter := o.mp.Meter(name)
	o.activeUsersGauge, err = meter.Int64ObservableGauge(
		name+".active.users.gauge",
		metric.WithDescription("Active Users Gauge"),
		metric.WithUnit("{users}"),
		metric.WithInt64Callback(func(_ context.Context, ob metric.Int64Observer) error {
			ob.Observe(o.activeUsersCount.Load())
			return nil
		}),
	)
	if err != nil {
		err = fmt.Errorf("init gauge: %v", err)
	}
	return
}

func (o *OtlpRecorder) ActiveUsersCountAdd(i int) {
	if o.activeUsersGauge == nil {
		if err := o.setupActiveUsersGauge(o.name); err != nil {
			log.Printf("setupActiveUsersGauge: %v", err)
			return
		}
	}
	o.activeUsersCount.Add(int64(i))
}

func WithRequestCount() OtlpMeterOption {
	return func(o *OtlpRecorder) error {
		return o.setupApiCounter(o.name)
	}
}

func (o *OtlpRecorder) setupApiCounter(name string) (err error) {
	meter := o.mp.Meter(name)
	o.apiCounter, err = meter.Int64Counter(
		name+".api.counter",
		metric.WithDescription("Number of API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		err = fmt.Errorf("int64Counter: %v", err)
	}
	return err
}

func (o *OtlpRecorder) CountApiRequest(ctx context.Context, i int, attrMap map[string]string) {
	if o.apiCounter == nil {
		if err := o.setupApiCounter(o.name); err != nil {
			log.Printf("setupApiCounter: %v", err)
			return
		}
	}
	attrs := make([]attribute.KeyValue, 0, len(attrMap))
	maps.DeleteFunc(attrMap, func(k, v string) bool { return k == "" || v == "" })
	for k, v := range attrMap {
		attrs = append(attrs, attribute.String(k, v))
	}
	o.apiCounter.Add(ctx, int64(i), metric.WithAttributes(attrs...))
}
