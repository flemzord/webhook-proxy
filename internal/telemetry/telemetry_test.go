package telemetry

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
)

func TestNewTracer(t *testing.T) {
	// Create a logger
	log := logrus.New()
	log.SetOutput(nil) // Silence logs during tests

	// Test with disabled config
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		ExporterType:   "stdout",
		Enabled:        false,
	}

	tracer, err := NewTracer(context.Background(), config, log)
	assert.NoError(t, err)
	assert.NotNil(t, tracer)
	assert.Equal(t, config, tracer.config)
	assert.Equal(t, log, tracer.log)

	// Test with enabled config
	config.Enabled = true
	tracer, err = NewTracer(context.Background(), config, log)
	assert.NoError(t, err)
	assert.NotNil(t, tracer)
	assert.Equal(t, config, tracer.config)
	assert.Equal(t, log, tracer.log)
}

func TestNewNoopTracer(t *testing.T) {
	tracer := NewNoopTracer()
	assert.NotNil(t, tracer)
	assert.False(t, tracer.config.Enabled)
}

func TestTracerShutdown(t *testing.T) {
	// Create a logger
	log := logrus.New()
	log.SetOutput(nil) // Silence logs during tests

	// Test with disabled config
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		ExporterType:   "stdout",
		Enabled:        false,
	}

	tracer, err := NewTracer(context.Background(), config, log)
	assert.NoError(t, err)

	// Shutdown should not error
	err = tracer.Shutdown(context.Background())
	assert.NoError(t, err)

	// Test with enabled config
	config.Enabled = true
	tracer, err = NewTracer(context.Background(), config, log)
	assert.NoError(t, err)

	// Shutdown should not error
	err = tracer.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestStartSpan(t *testing.T) {
	// Create a logger
	log := logrus.New()
	log.SetOutput(nil) // Silence logs during tests

	// Create a tracer
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		ExporterType:   "stdout",
		Enabled:        true,
	}

	tracer, err := NewTracer(context.Background(), config, log)
	assert.NoError(t, err)

	// Start a span
	ctx, span := tracer.StartSpan(context.Background(), "test-span")
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)

	// End the span
	span.End()
}

func TestHelperFunctions(t *testing.T) {
	// Create a logger
	log := logrus.New()
	log.SetOutput(nil) // Silence logs during tests

	// Create a tracer
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		ExporterType:   "stdout",
		Enabled:        true,
	}

	tracer, err := NewTracer(context.Background(), config, log)
	assert.NoError(t, err)

	// Start a span
	ctx, span := tracer.StartSpan(context.Background(), "test-span")

	// Test AddAttribute
	AddAttribute(ctx, "string-key", "string-value")
	AddAttribute(ctx, "int-key", 123)
	AddAttribute(ctx, "int64-key", int64(123))
	AddAttribute(ctx, "float64-key", 123.456)
	AddAttribute(ctx, "bool-key", true)
	AddAttribute(ctx, "nil-key", nil)
	AddAttribute(ctx, "struct-key", struct{ Name string }{"test"})

	// Test AddEvent
	AddEvent(ctx, "test-event", map[string]interface{}{
		"string-key":  "string-value",
		"int-key":     123,
		"int64-key":   int64(123),
		"float64-key": 123.456,
		"bool-key":    true,
		"nil-key":     nil,
		"struct-key":  struct{ Name string }{"test"},
	})

	// Test RecordError
	RecordError(ctx, errors.New("test-error"))

	// Test SetStatus
	SetStatus(ctx, codes.Error, "test-error")

	// Test SpanFromContext
	spanFromCtx := SpanFromContext(ctx)
	assert.Equal(t, span, spanFromCtx)

	// Test ContextWithSpan
	newCtx := ContextWithSpan(context.Background(), span)
	assert.NotNil(t, newCtx)
	assert.Equal(t, span, SpanFromContext(newCtx))

	// Test WithSpan
	err = WithSpan(ctx, "test-with-span", func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)

	err = WithSpan(ctx, "test-with-span-error", func(ctx context.Context) error {
		return errors.New("test-error")
	})
	assert.Error(t, err)
	assert.Equal(t, "test-error", err.Error())

	// End the span
	span.End()
}

func TestToString(t *testing.T) {
	assert.Equal(t, "", toString(nil))
	assert.Equal(t, "test", toString("test"))
	assert.Equal(t, "123", toString(123))
	assert.Equal(t, "123.456", toString(123.456))
	assert.Equal(t, "true", toString(true))
	assert.Equal(t, "{test}", toString(struct{ Name string }{"test"}))
}
