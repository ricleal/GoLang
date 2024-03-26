package main

import (
	"log/slog"
	"os"
	"time"

	"exp/binary-protocol/poetry/sample"
	"exp/binary-protocol/poetry/slam"

	"github.com/lmittmann/tint"
)

func Logger() *slog.Logger {
	w := os.Stderr
	logger := slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.TimeOnly,
		}),
	)
	return logger
}

func main() {
	l := Logger()

	l.Info("Running sample")
	sample.Run()

	l.Info("Running slam")
	slam.Run()

	l.Info("Done")
}
