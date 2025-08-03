package logger

import (
	"context"
	"io"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

type Fields map[string]any

type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...any)
	Info(msg string)
	Infof(format string, args ...any)
	Warn(msg string)
	Warnf(format string, args ...any)
	Error(msg string)
	Errorf(format string, args ...any)
	Fatal(msg string)
	Fatalf(format string, args ...any)

	WithField(key string, value any) Logger
	WithFields(fields Fields) Logger
	WithContext(ctx context.Context) Logger

	SetLevel(level Level)
	SetOutput(output io.Writer)
}
