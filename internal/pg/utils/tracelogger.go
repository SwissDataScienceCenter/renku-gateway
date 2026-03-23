package utils

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
)

// GetTraceLogger returns a pgx tracer which logs database operations via log/slog
func GetTraceLogger(level slog.Level) pgx.QueryTracer {
	return &tracelog.TraceLog{
		Logger:   &traceLogger{},
		LogLevel: toTracelogLevel(level),
	}
}

type traceLogger struct {
}

func (tl *traceLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	attrs := []slog.Attr{{Key: "Lvl", Value: slog.StringValue(level.String())}}
	for key, value := range data {
		attrs = append(attrs, slog.Attr{Key: key, Value: slog.AnyValue(value)})
	}
	slog.Default().LogAttrs(ctx, toSlogLevel(level), msg, attrs...)
}

func toSlogLevel(level tracelog.LogLevel) slog.Level {
	switch level {
	case tracelog.LogLevelError:
		return slog.LevelError
	case tracelog.LogLevelWarn:
		return slog.LevelWarn
	case tracelog.LogLevelInfo:
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}

func toTracelogLevel(level slog.Level) tracelog.LogLevel {
	switch {
	case level < slog.LevelInfo:
		return tracelog.LogLevelDebug
	case level < slog.LevelWarn:
		return tracelog.LogLevelInfo
	case level < slog.LevelError:
		return tracelog.LogLevelWarn
	default:
		return tracelog.LogLevelError
	}
}

// Check that TraceLogger satisfies the tracelog.Logger interface
var _ tracelog.Logger = (*traceLogger)(nil)
