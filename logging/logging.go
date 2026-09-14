package logging

import (
	"context"
	"reflect"

	"github.com/mitchellh/reflectwalk"
)

type Logger interface {
	Close(context.Context) error
	LogAccumulator() LogAccumulator
	Warn(msg string, metadata Metadata)
	Error(msg string, metadata Metadata)
	Info(msg string, metadata Metadata)
}

type LogAccumulator interface {
	AddMetadata(Metadata)
	Metadata() Metadata
}

type Metadata map[string][]string

type metadataWalker struct {
	metadata Metadata
}

func (w *metadataWalker) Struct(reflect.Value) error { return nil }

func (w *metadataWalker) StructField(
	field reflect.StructField,
	value reflect.Value,
) error {
	tag := field.Tag.Get("log")
	if tag == "" {
		return nil
	}
	value = reflect.Indirect(value)
	switch value.Kind() {
	case reflect.String:
		if value.String() != "" {
			w.metadata[tag] =
				append(w.metadata[tag], value.String())
		}
	case reflect.Slice:
		for i := range value.Len() {
			elem := value.Index(i)
			if elem.Kind() != reflect.String || elem.String() == "" {
				continue
			}
			w.metadata[tag] =
				append(w.metadata[tag], elem.String())
		}
	}
	return nil
}

func FromTagMetadata(a any) (m Metadata, err error) {
	w := &metadataWalker{
		metadata: make(map[string][]string),
	}
	err = reflectwalk.Walk(a, w)
	return w.metadata, err
}

func MockLogger() Logger { return &mockLogger{} }

type mockLogger struct{}

func (l *mockLogger) Close(context.Context) error         { return nil }
func (l *mockLogger) Warn(msg string, metadata Metadata)  {}
func (l *mockLogger) Error(msg string, metadata Metadata) {}
func (l *mockLogger) Info(msg string, metadata Metadata)  {}
func (l *mockLogger) AddMetadata(Metadata) error          { return nil }
func (l *mockLogger) Metadata() Metadata                  { return Metadata{} }
func (l *mockLogger) LogAccumulator() LogAccumulator {
	return &dFLogAccumulator{m: make(Metadata)}
}
