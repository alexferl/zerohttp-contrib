// Package tracer provides OpenTelemetry tracing middleware for zerohttp.
//
// This middleware integrates OpenTelemetry with zerohttp, enabling
// distributed tracing of HTTP requests for observability and debugging.
//
// Features:
//   - OpenTelemetry trace context propagation
//   - Automatic span creation for incoming requests
//   - OTLP HTTP exporter support
//   - Configurable sampling and span attributes
//
// Installation:
//
//	go get github.com/alexferl/zerohttp-contrib/middleware/tracer
//
// See https://opentelemetry.io for more information about OpenTelemetry.
package tracer
