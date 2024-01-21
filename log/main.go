package main

import (
	"context"

	"go.uber.org/zap"

	"exp/log/logger"
)

type key struct{}

// Example of a log passed in context
// see: https://www.kaznacheev.me/posts/en/where-to-place-logger-in-golang/
func main() {
	ctx := context.Background()

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	zLogger := zap.Must(
		config.Build(
			zap.AddCaller(),
			zap.AddStacktrace(zap.ErrorLevel),
		),
	)
	ctx = logger.ContextWithLogger(ctx, zLogger)

	zLogger.Info("Hello world")

	{
		childCtx := context.WithValue(ctx, key{}, "value")
		childCtx = logger.ContextWithLogger(childCtx, zLogger.With(zap.String("key", "value")))
		childZLogger := logger.LoggerFromContext(childCtx)
		childZLogger.Info("Hello world 2")

	}

	zlogger := logger.LoggerFromContext(ctx)
	zlogger.Error("Hello world 3")
	zLogger.Debug("Hello world 4")
}
