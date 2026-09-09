package metering

import (
	"context"
	"time"
)

type SetupFunc func(name string) error

type Meter interface {
	Close(context.Context) error
	MemoryUsage(name string) error
	Uptime(name string) error
	RecordLatency(ctx context.Context, dur time.Duration) error
	ActiveUsersCountAdd(i int)
	CountApiRequest(ctx context.Context, i int, attr map[string]string)
}

type MockMeter struct{}

func (m *MockMeter) Close(context.Context) error { return nil }

func (m *MockMeter) MemoryUsage(name string) error { return nil }

func (m *MockMeter) Uptime(name string) error { return nil }

func (m *MockMeter) RecordLatency(ctx context.Context, dur time.Duration) error { return nil }

func (m *MockMeter) ActiveUsersCountAdd(i int) {}

func (m *MockMeter) CountApiRequest(ctx context.Context, i int, attr map[string]string) {}
