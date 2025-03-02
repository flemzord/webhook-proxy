// Package telemetry provides OpenTelemetry tracing functionality
package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Config represents the configuration for telemetry
type Config struct {
	ServiceName    string
	ServiceVersion string
	ExporterType   string // stdout, otlp, etc.
	Endpoint       string // for OTLP exporter
	Enabled        bool
}

// Tracer is a wrapper around the OpenTelemetry tracer
type Tracer struct {
	tracer trace.Tracer
	log    *logrus.Logger
	config Config
}

// NewTracer creates a new tracer with the given configuration
func NewTracer(ctx context.Context, config Config, log *logrus.Logger) (*Tracer, error) {
	if !config.Enabled {
		return &Tracer{
			tracer: trace.NewNoopTracerProvider().Tracer("noop"),
			log:    log,
			config: config,
		}, nil
	}

	// Create exporter
	var exporter sdktrace.SpanExporter
	var err error

	switch config.ExporterType {
	case "stdout":
		exporter, err = stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
	default:
		// Default to stdout
		exporter, err = stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
	}

	if err != nil {
		return nil, err
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Create tracer
	tracer := tp.Tracer(config.ServiceName)

	return &Tracer{
		tracer: tracer,
		log:    log,
		config: config,
	}, nil
}

// Shutdown shuts down the tracer provider
func (t *Tracer) Shutdown(ctx context.Context) error {
	if !t.config.Enabled {
		return nil
	}

	// Get tracer provider
	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	if !ok {
		t.log.Warn("Failed to get tracer provider for shutdown")
		return nil
	}

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := tp.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

// StartSpan starts a new span with the given name
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name)
}

// AddAttribute adds an attribute to the current span
func AddAttribute(ctx context.Context, key string, value interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	var attr attribute.KeyValue
	switch v := value.(type) {
	case string:
		attr = attribute.String(key, v)
	case int:
		attr = attribute.Int(key, v)
	case int64:
		attr = attribute.Int64(key, v)
	case float64:
		attr = attribute.Float64(key, v)
	case bool:
		attr = attribute.Bool(key, v)
	default:
		// Try to convert to string
		attr = attribute.String(key, toString(value))
	}

	span.SetAttributes(attr)
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attributes map[string]interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	attrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		switch val := v.(type) {
		case string:
			attrs = append(attrs, attribute.String(k, val))
		case int:
			attrs = append(attrs, attribute.Int(k, val))
		case int64:
			attrs = append(attrs, attribute.Int64(k, val))
		case float64:
			attrs = append(attrs, attribute.Float64(k, val))
		case bool:
			attrs = append(attrs, attribute.Bool(k, val))
		default:
			// Try to convert to string
			attrs = append(attrs, attribute.String(k, toString(v)))
		}
	}

	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// RecordError records an error in the current span
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.RecordError(err)
}

// SetStatus sets the status of the current span
func SetStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.SetStatus(code, description)
}

// toString converts a value to string
func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	if s, ok := value.(fmt.Stringer); ok {
		return s.String()
	}
	return fmt.Sprintf("%v", value)
}

// NewNoopTracer creates a new noop tracer
func NewNoopTracer() *Tracer {
	return &Tracer{
		tracer: trace.NewNoopTracerProvider().Tracer("noop"),
		log:    logrus.New(),
		config: Config{Enabled: false},
	}
}

// WithSpan wraps a function with a span
func WithSpan(ctx context.Context, name string, fn func(context.Context) error) error {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, name)
	defer span.End()

	return fn(ctx)
}

// SpanFromContext returns the current span from the context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ContextWithSpan returns a new context with the given span
func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}
