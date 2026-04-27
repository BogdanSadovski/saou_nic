package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	defaultLogger *zap.Logger
	sugarLogger   *zap.SugaredLogger
	initOnce      sync.Once
)

// Init initializes the global logger with the given configuration.
// This should be called once at application startup.
func Init(cfg LoggerConfig) error {
	var err error
	initOnce.Do(func() {
		defaultLogger, sugarLogger, err = newLogger(cfg)
	})
	return err
}

// InitWithZap initializes the global logger with a pre-configured zap.Logger.
func InitWithZap(l *zap.Logger) {
	initOnce.Do(func() {
		defaultLogger = l
		sugarLogger = l.Sugar()
	})
}

// newLogger creates a new zap.Logger and SugaredLogger based on the config.
func newLogger(cfg LoggerConfig) (*zap.Logger, *zap.SugaredLogger, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	var writeSyncer zapcore.WriteSyncer
	switch cfg.Output {
	case "stderr":
		writeSyncer = zapcore.Lock(os.Stderr)
	case "file":
		if cfg.OutputPath == "" {
			writeSyncer = zapcore.Lock(os.Stdout)
		} else {
			f, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, nil, err
			}
			writeSyncer = zapcore.Lock(f)
		}
	default:
		writeSyncer = zapcore.Lock(os.Stdout)
	}

	core := zapcore.NewCore(encoder, writeSyncer, level)

	opts := []zap.Option{
		zap.AddCaller(),
	}
	if cfg.DisableCaller {
		opts = append(opts, zap.WithCaller(false))
	}
	if !cfg.DisableStacktrace {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}
	if cfg.Development {
		opts = append(opts, zap.Development())
	}

	logger := zap.New(core, opts...)
	return logger, logger.Sugar(), nil
}

// Get returns the default zap.Logger.
func Get() *zap.Logger {
	if defaultLogger == nil {
		cfg := DefaultLoggerConfig()
		l, s, _ := newLogger(cfg)
		defaultLogger = l
		sugarLogger = s
	}
	return defaultLogger
}

// GetSugar returns the default SugaredLogger for less structured logging.
func GetSugar() *zap.SugaredLogger {
	if sugarLogger == nil {
		_ = Init(DefaultLoggerConfig())
	}
	return sugarLogger
}

// Debug logs a message at DEBUG level.
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info logs a message at INFO level.
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn logs a message at WARN level.
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error logs a message at ERROR level.
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal logs a message at FATAL level and exits.
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// Panic logs a message at PANIC level and panics.
func Panic(msg string, fields ...zap.Field) {
	Get().Panic(msg, fields...)
}

// With returns a logger with the given fields attached.
func With(fields ...zap.Field) *zap.Logger {
	return Get().With(fields...)
}

// Sync flushes any buffered log entries.
func Sync() error {
	if defaultLogger != nil {
		return defaultLogger.Sync()
	}
	return nil
}
