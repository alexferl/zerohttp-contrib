package zerolog

import (
	"context"
	"os"

	"github.com/alexferl/zerohttp/log"
	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger to implement zerohttp's log.Logger interface.
// It provides high-performance, zero-allocation JSON structured logging.
type Logger struct {
	logger zerolog.Logger
}

// Ensure Logger implements log.Logger
var _ log.Logger = (*Logger)(nil)

// New creates a new zerolog logger with the provided zerolog instance.
func New(logger zerolog.Logger) *Logger {
	return &Logger{logger: logger}
}

// NewDefault creates a zerolog logger with sensible defaults.
// It outputs JSON to stdout with timestamps.
func NewDefault() *Logger {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &Logger{logger: logger}
}

// NewConsole creates a zerolog logger with console-friendly output.
// Useful for development - outputs human-readable format instead of JSON.
func NewConsole() *Logger {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	return &Logger{logger: logger}
}

// NewConsoleWithLevel creates a console logger with a specific log level.
func NewConsoleWithLevel(level zerolog.Level) *Logger {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
		Level(level).
		With().
		Timestamp().
		Logger()
	return &Logger{logger: logger}
}

// Debug logs a debug message with optional fields.
func (l *Logger) Debug(msg string, fields ...log.Field) {
	event := l.logger.Debug()
	l.addFields(event, fields...)
	event.Msg(msg)
}

// Info logs an info message with optional fields.
func (l *Logger) Info(msg string, fields ...log.Field) {
	event := l.logger.Info()
	l.addFields(event, fields...)
	event.Msg(msg)
}

// Warn logs a warning message with optional fields.
func (l *Logger) Warn(msg string, fields ...log.Field) {
	event := l.logger.Warn()
	l.addFields(event, fields...)
	event.Msg(msg)
}

// Error logs an error message with optional fields.
func (l *Logger) Error(msg string, fields ...log.Field) {
	event := l.logger.Error()
	l.addFields(event, fields...)
	event.Msg(msg)
}

// Panic logs a panic message with optional fields and then panics.
func (l *Logger) Panic(msg string, fields ...log.Field) {
	event := l.logger.Panic()
	l.addFields(event, fields...)
	event.Msg(msg)
}

// Fatal logs a fatal message with optional fields and then exits with code 1.
func (l *Logger) Fatal(msg string, fields ...log.Field) {
	event := l.logger.Fatal()
	l.addFields(event, fields...)
	event.Msg(msg)
}

// WithFields returns a new logger with additional fields.
// The fields are added to all subsequent log entries.
func (l *Logger) WithFields(fields ...log.Field) log.Logger {
	ctx := l.logger.With()
	for _, field := range fields {
		ctx = ctx.Interface(field.Key, field.Value)
	}
	return &Logger{logger: ctx.Logger()}
}

// WithContext returns a new logger with context.
// The context is added to all subsequent log entries for tracing.
func (l *Logger) WithContext(ctx context.Context) log.Logger {
	return &Logger{logger: l.logger.With().Ctx(ctx).Logger()}
}

// addFields adds structured fields to a zerolog event.
// It handles common types efficiently and falls back to interface{} for others.
func (l *Logger) addFields(event *zerolog.Event, fields ...log.Field) {
	for _, field := range fields {
		switch v := field.Value.(type) {
		case error:
			event.Err(v)
		case string:
			event.Str(field.Key, v)
		case int:
			event.Int(field.Key, v)
		case int8:
			event.Int8(field.Key, v)
		case int16:
			event.Int16(field.Key, v)
		case int32:
			event.Int32(field.Key, v)
		case int64:
			event.Int64(field.Key, v)
		case uint:
			event.Uint(field.Key, v)
		case uint8:
			event.Uint8(field.Key, v)
		case uint16:
			event.Uint16(field.Key, v)
		case uint32:
			event.Uint32(field.Key, v)
		case uint64:
			event.Uint64(field.Key, v)
		case float32:
			event.Float32(field.Key, v)
		case float64:
			event.Float64(field.Key, v)
		case bool:
			event.Bool(field.Key, v)
		case []byte:
			event.Bytes(field.Key, v)
		case []string:
			event.Strs(field.Key, v)
		case []int:
			event.Ints(field.Key, v)
		case []int64:
			event.Ints64(field.Key, v)
		default:
			event.Interface(field.Key, v)
		}
	}
}

// Unwrap returns the underlying zerolog.Logger.
// This allows direct access to zerolog functionality if needed.
func (l *Logger) Unwrap() zerolog.Logger {
	return l.logger
}
