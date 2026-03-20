package zerolog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	zl := zerolog.New(os.Stdout)
	logger := New(zl)
	assert.NotNil(t, logger)
}

func TestNewDefault(t *testing.T) {
	logger := NewDefault()
	assert.NotNil(t, logger)
}

func TestNewConsole(t *testing.T) {
	logger := NewConsole()
	assert.NotNil(t, logger)
}

func TestNewConsoleWithLevel(t *testing.T) {
	logger := NewConsoleWithLevel(zerolog.WarnLevel)
	assert.NotNil(t, logger)
}

func TestLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf).Level(zerolog.DebugLevel)
	logger := New(zl)

	logger.Debug("debug message", log.F("key", "value"))

	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, `"key":"value"`)
	assert.Contains(t, output, `"level":"debug"`)
}

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf)
	logger := New(zl)

	logger.Info("info message", log.F("count", 42))

	output := buf.String()
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, `"count":42`)
	assert.Contains(t, output, `"level":"info"`)
}

func TestLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf)
	logger := New(zl)

	logger.Warn("warn message", log.F("threshold", 0.75))

	output := buf.String()
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, `"threshold":0.75`)
	assert.Contains(t, output, `"level":"warn"`)
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf)
	logger := New(zl)

	testErr := errors.New("something went wrong")
	logger.Error("error message", log.E(testErr))

	output := buf.String()
	assert.Contains(t, output, "error message")
	assert.Contains(t, output, `"error":"something went wrong"`)
	assert.Contains(t, output, `"level":"error"`)
}

func TestLogger_Panic(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf)
	logger := New(zl)

	assert.Panics(t, func() {
		logger.Panic("panic message", log.F("reason", "test"))
	})

	output := buf.String()
	assert.Contains(t, output, "panic message")
	assert.Contains(t, output, `"reason":"test"`)
	assert.Contains(t, output, `"level":"panic"`)
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf)
	logger := New(zl)

	loggerWithFields := logger.WithFields(
		log.F("service", "test-service"),
		log.F("version", "1.0.0"),
	)

	loggerWithFields.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, `"service":"test-service"`)
	assert.Contains(t, output, `"version":"1.0.0"`)
}

func TestLogger_WithContext(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf)
	logger := New(zl)

	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("request_id"), "abc123")
	loggerWithCtx := logger.WithContext(ctx)

	loggerWithCtx.Info("context test")

	output := buf.String()
	assert.Contains(t, output, "context test")
}

func TestLogger_FieldTypes(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf)
	logger := New(zl)

	logger.Info("type test",
		log.F("string", "value"),
		log.F("int", 42),
		log.F("int8", int8(8)),
		log.F("int16", int16(16)),
		log.F("int32", int32(32)),
		log.F("int64", int64(64)),
		log.F("uint", uint(42)),
		log.F("uint8", uint8(8)),
		log.F("uint16", uint16(16)),
		log.F("uint32", uint32(32)),
		log.F("uint64", uint64(64)),
		log.F("float32", float32(3.14)),
		log.F("float64", 2.718),
		log.F("bool", true),
		log.F("bytes", []byte("hello")),
		log.F("strings", []string{"a", "b", "c"}),
		log.F("ints", []int{1, 2, 3}),
		log.F("int64s", []int64{1, 2, 3}),
		log.F("interface", map[string]any{"nested": "value"}),
	)

	output := buf.String()

	// Verify all fields are present
	var result map[string]any
	err := json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	assert.Equal(t, "value", result["string"])
	assert.Equal(t, float64(42), result["int"])
	assert.Equal(t, float64(8), result["int8"])
	assert.Equal(t, float64(16), result["int16"])
	assert.Equal(t, float64(32), result["int32"])
	assert.Equal(t, float64(64), result["int64"])
	assert.Equal(t, float64(42), result["uint"])
	assert.Equal(t, float64(8), result["uint8"])
	assert.Equal(t, float64(16), result["uint16"])
	assert.Equal(t, float64(32), result["uint32"])
	assert.Equal(t, float64(64), result["uint64"])
	assert.Equal(t, float64(3.14), result["float32"])
	assert.Equal(t, 2.718, result["float64"])
	assert.Equal(t, true, result["bool"])
	assert.Equal(t, "hello", result["bytes"])
	assert.Contains(t, result, "strings")
	assert.Contains(t, result, "ints")
	assert.Contains(t, result, "int64s")
	assert.Contains(t, result, "interface")
}

func TestLogger_Unwrap(t *testing.T) {
	zl := zerolog.New(os.Stdout)
	logger := New(zl)

	unwrapped := logger.Unwrap()
	assert.Equal(t, zl, unwrapped)
}

func TestLogger_Interface(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf)
	logger := New(zl)

	// Verify it implements log.Logger
	var _ log.Logger = logger

	// Test all methods exist and work
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, 4)
}

func TestLogger_ChainedFields(t *testing.T) {
	var buf bytes.Buffer
	zl := zerolog.New(&buf)
	logger := New(zl)

	// Chain multiple WithFields calls
	logger1 := logger.WithFields(log.F("layer", "1"))
	logger2 := logger1.WithFields(log.F("layer", "2"))
	logger3 := logger2.WithFields(log.F("layer", "3"))

	logger3.Info("chained")

	output := buf.String()
	// The last WithFields wins for the same key
	assert.Contains(t, output, `"layer":"3"`)
}

func TestNewConsole_Output(t *testing.T) {
	// Console logger should produce human-readable output
	logger := NewConsoleWithLevel(zerolog.InfoLevel)
	assert.NotNil(t, logger)

	// Just verify it doesn't panic and is properly configured
	logger.Info("console test", log.F("key", "value"))
}
