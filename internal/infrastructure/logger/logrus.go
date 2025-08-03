package logger

import (
	"context"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

type logrusLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

func NewLogrusLogger(config *Config) Logger {
	logger := logrus.New()

	// Set level
	switch config.Level {
	case LevelDebug:
		logger.SetLevel(logrus.DebugLevel)
	case LevelInfo:
		logger.SetLevel(logrus.InfoLevel)
	case LevelWarn:
		logger.SetLevel(logrus.WarnLevel)
	case LevelError:
		logger.SetLevel(logrus.ErrorLevel)
	case LevelFatal:
		logger.SetLevel(logrus.FatalLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set formatter
	switch config.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "caller",
			},
		})
	case "console":
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			ForceColors:     true,
			DisableColors:   false,
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			FullTimestamp:   true,
			DisableColors:   true,
		})
	default:
		// Default to console for better development experience
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			ForceColors:     true,
			DisableColors:   false,
		})
	}

	// Set output
	switch config.Output {
	case "stderr":
		logger.SetOutput(os.Stderr)
	case "file":
		if config.FilePath != "" {
			logger.SetOutput(&lumberjack.Logger{
				Filename:   config.FilePath,
				MaxSize:    config.MaxSize,
				MaxBackups: config.MaxBackups,
				MaxAge:     config.MaxAge,
				Compress:   config.Compress,
			})
		} else {
			logger.SetOutput(os.Stdout)
		}
	default:
		logger.SetOutput(os.Stdout)
	}

	// Add static fields for container environments
	fields := logrus.Fields{}
	for k, v := range config.Fields {
		fields[k] = v
	}

	baseEntry := logrus.NewEntry(logger).WithFields(fields)

	return &logrusLogger{
		logger: logger,
		entry:  baseEntry,
	}
}

func (l *logrusLogger) Debug(msg string)                  { l.entry.Debug(msg) }
func (l *logrusLogger) Debugf(format string, args ...any) { l.entry.Debugf(format, args...) }
func (l *logrusLogger) Info(msg string)                   { l.entry.Info(msg) }
func (l *logrusLogger) Infof(format string, args ...any)  { l.entry.Infof(format, args...) }
func (l *logrusLogger) Warn(msg string)                   { l.entry.Warn(msg) }
func (l *logrusLogger) Warnf(format string, args ...any)  { l.entry.Warnf(format, args...) }
func (l *logrusLogger) Error(msg string)                  { l.entry.Error(msg) }
func (l *logrusLogger) Errorf(format string, args ...any) { l.entry.Errorf(format, args...) }
func (l *logrusLogger) Fatal(msg string)                  { l.entry.Fatal(msg) }
func (l *logrusLogger) Fatalf(format string, args ...any) { l.entry.Fatalf(format, args...) }

func (l *logrusLogger) WithField(key string, value interface{}) Logger {
	return &logrusLogger{
		entry: l.entry.WithField(key, value),
	}
}

func (l *logrusLogger) WithFields(fields Fields) Logger {
	return &logrusLogger{
		entry: l.entry.WithFields(logrus.Fields(fields)),
	}
}

func (l *logrusLogger) WithContext(ctx context.Context) Logger {
	return &logrusLogger{
		entry: l.entry.WithContext(ctx),
	}
}

func (l *logrusLogger) SetLevel(level Level) {
	switch level {
	case LevelDebug:
		l.logger.SetLevel(logrus.DebugLevel)
	case LevelInfo:
		l.logger.SetLevel(logrus.InfoLevel)
	case LevelWarn:
		l.logger.SetLevel(logrus.WarnLevel)
	case LevelError:
		l.logger.SetLevel(logrus.ErrorLevel)
	case LevelFatal:
		l.logger.SetLevel(logrus.FatalLevel)
	}
}

func (l *logrusLogger) SetOutput(output io.Writer) {
	l.logger.SetOutput(output)
}
