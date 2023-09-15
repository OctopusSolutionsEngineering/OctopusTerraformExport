package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

// BuildLogger ensures everything printed by the logger is done to stderr.
// This allows the application to print HCL to stdout, which can be redirected to a file.
// But application messages are kept in a separate stream.
func BuildLogger() {
	allLevels := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return true
	})

	// write syncers
	stderrSyncer := zapcore.Lock(os.Stderr)

	// tee core
	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
				TimeKey:        "ts",
				LevelKey:       "level",
				NameKey:        "logger",
				CallerKey:      "caller",
				FunctionKey:    zapcore.OmitKey,
				MessageKey:     "msg",
				StacktraceKey:  "stacktrace",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.LowercaseLevelEncoder,
				EncodeTime:     zapcore.ISO8601TimeEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			}),
			stderrSyncer,
			allLevels,
		),
	)

	// finally construct the logger with the tee core
	logger := zap.New(core)

	// replace the global logger
	zap.ReplaceGlobals(logger)
}
