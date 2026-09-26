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

func MockMeter() Meter { return &mockMeter{} }

type mockMeter struct{}

func (m *mockMeter) Close(context.Context) error { return nil }

func (m *mockMeter) MemoryUsage(name string) error { return nil }

func (m *mockMeter) Uptime(name string) error { return nil }

func (m *mockMeter) RecordLatency(ctx context.Context, dur time.Duration) error { return nil }

func (m *mockMeter) ActiveUsersCountAdd(i int) {}

func (m *mockMeter) CountApiRequest(ctx context.Context, i int, attr map[string]string) {}
