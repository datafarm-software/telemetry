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

type OtlpRecorder struct {
	requestLatency                             metric.Float64Histogram
	apiCounter                                 metric.Int64Counter
	memoryGauge, uptimeGauge, activeUsersGauge metric.Int64ObservableGauge
	name                                       string
	mp                                         *sdkmetric.MeterProvider
	activeUsersCount                           atomic.Int64
}

type OtlpOpts struct {
	Name, Endpoint      string
	Res                 *resource.Resource
	MemoryUsage, Uptime bool
}

// NOTE: This uses insecure HTTP
func NewOtlpMeter(opts OtlpOpts) (
	Meter, error) {
	exporter, err := otlpmetrichttp.New(
		context.Background(), otlpmetrichttp.WithEndpoint(opts.Endpoint),
		otlpmetrichttp.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("init exporter: %v", err)
	}
	reader := sdkmetric.NewPeriodicReader(exporter,
		sdkmetric.WithProducer(otelruntime.NewProducer()))
	sdkOpts := []sdkmetric.Option{sdkmetric.WithReader(reader)}
	if opts.Res != nil {
		sdkOpts = append(sdkOpts, sdkmetric.WithResource(opts.Res))
	}
	mp := sdkmetric.NewMeterProvider(sdkOpts...)
	o := &OtlpRecorder{name: opts.Name, mp: mp}
	if opts.MemoryUsage {
		if err = o.MemoryUsage(o.name); err != nil {
			return nil, fmt.Errorf("memory usage: %v", err)
		}
	}
	if opts.Uptime {
		if err = o.Uptime(o.name); err != nil {
			return nil, fmt.Errorf("uptime: %v", err)
		}
	}
	return o, nil
}

func (o *OtlpRecorder) Close(ctx context.Context) error {
	return o.mp.Shutdown(ctx)
}

func (o *OtlpRecorder) DefaultSetup(name string) (err error) {
	if err = o.OtelRunTime(name); err != nil {
		return fmt.Errorf("otel runtime: %v", err)
	}
	return nil
}

func (o *OtlpRecorder) setupOtelRunTime(_ string) (err error) {
	return otelruntime.Start()
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

var processStart = time.Now().Unix()

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
		return fmt.Errorf("init gauge: %v", err)
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

func (o *OtlpRecorder) ActiveUsersCountAdd(i int) {
	if o.activeUsersGauge == nil {
		if err := o.setupActiveUsersGauge(o.name); err != nil {
			log.Printf("setupActiveUsersGauge: %v", err)
			return
		}
	}
	o.activeUsersCount.Add(int64(i))
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
