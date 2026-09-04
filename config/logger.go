package config

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/lumberjack.v2"
)

func parseLevel(val string) slog.Level {
	switch strings.ToLower(val) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func NewLogger() *slog.Logger {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		panic("can't make logs directory: " + err.Error())
	}

	rotator := &lumberjack.Logger{
		Filename:   filepath.Join("logs", "app.log"),
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     14,
		Compress:   true,
	}

	writer := io.MultiWriter(os.Stdout, rotator)
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: parseLevel(GetEnv("LOG_LEVEL", "info")),
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
