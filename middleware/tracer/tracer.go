package tracer

import (
	"context"
	"fmt"

	"github.com/alexferl/zerohttp/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	otelTrace "go.opentelemetry.io/otel/trace"
)

// OTelTracer wraps OpenTelemetry tracer for zerohttp.
type OTelTracer struct {
	tracer otelTrace.Tracer
}

// NewOTelTracer creates a new OTelTracer wrapping the provided OpenTelemetry tracer.
func NewOTelTracer(tracer otelTrace.Tracer) *OTelTracer {
	return &OTelTracer{tracer: tracer}
}

// NewDefault creates a new OTelTracer with default OTLP HTTP exporter setup.
// It configures a tracer provider with the given service name and endpoint.
// Returns the tracer, a shutdown function to flush and close the provider, and any error.
func NewDefault(ctx context.Context, serviceName, endpoint string) (*OTelTracer, func(), error) {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(provider)

	tracer := provider.Tracer("zerohttp")
	shutdown := func() {
		_ = provider.Shutdown(ctx)
	}

	return NewOTelTracer(tracer), shutdown, nil
}

// Start creates a new span and returns the updated context.
func (t *OTelTracer) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
	ctx, otelSpan := t.tracer.Start(ctx, name)
	return ctx, &OTelSpan{span: otelSpan}
}

// OTelSpan wraps an OpenTelemetry span for zerohttp.
type OTelSpan struct {
	span otelTrace.Span
}

// End completes the span.
func (s *OTelSpan) End() { s.span.End() }

// SetStatus sets the status of the span.
func (s *OTelSpan) SetStatus(code trace.Code, desc string) { s.span.SetStatus(toOtelCode(code), desc) }

// RecordError records an error as an exception on the span.
func (s *OTelSpan) RecordError(err error, opts ...trace.ErrorOption) { s.span.RecordError(err) }

// SetAttributes adds attributes to the span.
func (s *OTelSpan) SetAttributes(attrs ...trace.Attribute) {
	for _, attr := range attrs {
		s.span.SetAttributes(toOtelAttribute(attr))
	}
}

func toOtelAttribute(attr trace.Attribute) attribute.KeyValue {
	switch v := attr.Value.(type) {
	case string:
		return attribute.String(attr.Key, v)
	case int:
		return attribute.Int(attr.Key, v)
	case int64:
		return attribute.Int64(attr.Key, v)
	case float64:
		return attribute.Float64(attr.Key, v)
	case bool:
		return attribute.Bool(attr.Key, v)
	default:
		return attribute.String(attr.Key, "")
	}
}

func toOtelCode(code trace.Code) codes.Code {
	switch code {
	case trace.CodeOk:
		return codes.Ok
	case trace.CodeError:
		return codes.Error
	default:
		return codes.Unset
	}
}
