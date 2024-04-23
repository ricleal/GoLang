package main

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

func Logger(w io.Writer, levelAsString string) *slog.Logger {
	var level slog.Level

	switch strings.ToLower(levelAsString) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "Error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      level,
			TimeFormat: time.TimeOnly,
		}),
	)

	return logger
}

type DoSomething func(string) string

func (f DoSomething) Prefixed(prefix string) DoSomething {
	return func(s string) string {
		return prefix + " :: " + f(s)
	}
}

func main() {
	log := Logger(os.Stderr, os.Getenv("LOG_LEVEL"))

	var ds DoSomething = strings.ToUpper // same as func(s string) string { return strings.ToUpper(s) }
	out1 := ds("ricardo")

	ds2 := ds.Prefixed("hello")
	out2 := ds2("world")

	log.Info("example 1", slog.Any("out1", out1))
	log.Info("example 2", slog.Any("out2", out2))
}
