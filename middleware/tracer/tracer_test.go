package tracer

import (
	"context"
	"errors"
	"testing"

	"github.com/alexferl/zerohttp/trace"
	"github.com/alexferl/zerohttp/zhtest"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewDefault(t *testing.T) {
	t.Run("creates tracer with valid endpoint", func(t *testing.T) {
		// Use a local endpoint that won't actually be called during this test
		ctx := context.Background()
		tracer, shutdown, err := NewDefault(ctx, "test-service", "localhost:4318")

		// The exporter is created synchronously but connection happens later
		// So this should succeed even if the endpoint isn't reachable
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotNil(t, tracer)
		zhtest.AssertNotNil(t, shutdown)

		// Clean up
		if shutdown != nil {
			shutdown()
		}
	})
}

func TestNewOTelTracer(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)

	otelTracer := provider.Tracer("test")
	tracer := NewOTelTracer(otelTracer)

	if tracer == nil {
		t.Fatal("expected tracer to not be nil")
	}
	if tracer.tracer == nil {
		t.Fatal("expected wrapped tracer to not be nil")
	}
}

func TestOTelTracer_Start(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)

	otelTracer := provider.Tracer("test")
	tracer := NewOTelTracer(otelTracer)

	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")

	if span == nil {
		t.Fatal("expected span to not be nil")
	}

	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "test-span" {
		t.Errorf("expected span name 'test-span', got %s", spans[0].Name)
	}
}

func TestOTelSpan_End(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)

	otelTracer := provider.Tracer("test")
	tracer := NewOTelTracer(otelTracer)

	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
}

func TestOTelSpan_SetStatus(t *testing.T) {
	tests := []struct {
		name     string
		code     trace.Code
		desc     string
		expected codes.Code
	}{
		{
			name:     "ok status",
			code:     trace.CodeOk,
			desc:     "success",
			expected: codes.Ok,
		},
		{
			name:     "error status",
			code:     trace.CodeError,
			desc:     "failure",
			expected: codes.Error,
		},
		{
			name:     "unset status",
			code:     trace.CodeUnset,
			desc:     "",
			expected: codes.Unset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := tracetest.NewInMemoryExporter()
			provider := sdktrace.NewTracerProvider(
				sdktrace.WithSyncer(exporter),
			)

			otelTracer := provider.Tracer("test")
			tracer := NewOTelTracer(otelTracer)

			ctx := context.Background()
			_, span := tracer.Start(ctx, "test-span")
			span.SetStatus(tt.code, tt.desc)
			span.End()

			spans := exporter.GetSpans()
			if len(spans) != 1 {
				t.Fatalf("expected 1 span, got %d", len(spans))
			}

			if spans[0].Status.Code != tt.expected {
				t.Errorf("expected status code %v, got %v", tt.expected, spans[0].Status.Code)
			}
		})
	}
}

func TestOTelSpan_RecordError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)

	otelTracer := provider.Tracer("test")
	tracer := NewOTelTracer(otelTracer)

	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")

	testErr := errors.New("test error")
	span.RecordError(testErr)
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	s := spans[0]
	if len(s.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(s.Events))
	}

	if s.Events[0].Name != "exception" {
		t.Errorf("expected event name 'exception', got %s", s.Events[0].Name)
	}
}

func TestOTelSpan_SetAttributes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)

	otelTracer := provider.Tracer("test")
	tracer := NewOTelTracer(otelTracer)

	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")

	span.SetAttributes(
		trace.String("string-attr", "value"),
		trace.Int("int-attr", 42),
		trace.Int64("int64-attr", 64),
		trace.Float64("float-attr", 3.14),
		trace.Bool("bool-attr", true),
	)
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	attrs := spans[0].Attributes
	expectedAttrs := map[string]attribute.Value{
		"string-attr": attribute.StringValue("value"),
		"int-attr":    attribute.Int64Value(42),
		"int64-attr":  attribute.Int64Value(64),
		"float-attr":  attribute.Float64Value(3.14),
		"bool-attr":   attribute.BoolValue(true),
	}

	for key, expected := range expectedAttrs {
		found := false
		for _, attr := range attrs {
			if string(attr.Key) == key {
				found = true
				if attr.Value != expected {
					t.Errorf("expected %s to be %v, got %v", key, expected, attr.Value)
				}
				break
			}
		}
		if !found {
			t.Errorf("expected attribute %s not found", key)
		}
	}
}

func TestToOtelAttribute(t *testing.T) {
	tests := []struct {
		name     string
		attr     trace.Attribute
		expected attribute.KeyValue
	}{
		{
			name:     "string",
			attr:     trace.String("key", "value"),
			expected: attribute.String("key", "value"),
		},
		{
			name:     "int",
			attr:     trace.Int("key", 42),
			expected: attribute.Int("key", 42),
		},
		{
			name:     "int64",
			attr:     trace.Int64("key", 64),
			expected: attribute.Int64("key", 64),
		},
		{
			name:     "float64",
			attr:     trace.Float64("key", 3.14),
			expected: attribute.Float64("key", 3.14),
		},
		{
			name:     "bool",
			attr:     trace.Bool("key", true),
			expected: attribute.Bool("key", true),
		},
		{
			name:     "default",
			attr:     trace.Attribute{Key: "key", Value: struct{}{}},
			expected: attribute.String("key", ""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toOtelAttribute(tt.attr)
			if result.Key != tt.expected.Key {
				t.Errorf("expected key %s, got %s", tt.expected.Key, result.Key)
			}
			if result.Value != tt.expected.Value {
				t.Errorf("expected value %v, got %v", tt.expected.Value, result.Value)
			}
		})
	}
}

func TestToOtelCode(t *testing.T) {
	tests := []struct {
		name     string
		code     trace.Code
		expected codes.Code
	}{
		{
			name:     "ok",
			code:     trace.CodeOk,
			expected: codes.Ok,
		},
		{
			name:     "error",
			code:     trace.CodeError,
			expected: codes.Error,
		},
		{
			name:     "unset",
			code:     trace.CodeUnset,
			expected: codes.Unset,
		},
		{
			name:     "unknown",
			code:     trace.Code(999),
			expected: codes.Unset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toOtelCode(tt.code)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
