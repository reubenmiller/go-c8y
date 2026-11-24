package api

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// restyLogger sends the HTTP layer's messages to a slog logger, so they
// follow the application's logging instead of always going to stderr.
//
// Its debug messages are only produced when ClientOptions.Debug is set, so
// they are logged at info level to be visible with a default logger. Its
// warnings are about how the HTTP layer is used (e.g. credentials over plain
// HTTP) and are also logged at info level, so an application can hide them
// without hiding the SDK's own warnings.
type restyLogger struct {
	logger *slog.Logger
}

func (l restyLogger) log(level slog.Level, format string, v ...any) {
	logger := l.logger
	if logger == nil {
		logger = slog.Default()
	}
	if !logger.Enabled(context.Background(), level) {
		return
	}
	logger.Log(context.Background(), level, strings.TrimRight(fmt.Sprintf(format, v...), "\n"), "component", "http")
}

func (l restyLogger) Errorf(format string, v ...any) { l.log(slog.LevelError, format, v...) }
func (l restyLogger) Warnf(format string, v ...any)  { l.log(slog.LevelInfo, format, v...) }
func (l restyLogger) Debugf(format string, v ...any) { l.log(slog.LevelInfo, format, v...) }
