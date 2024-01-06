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

	zLogger := zap.Must(zap.NewProduction())
	ctx = logger.ContextWithLogger(ctx, zLogger)

	zLogger.Info("Hello world")

	{
		childCtx := context.WithValue(ctx, key{}, "value")
		childZLogger := zLogger.With(zap.String("key", "value"))
		childCtx = logger.ContextWithLogger(childCtx, childZLogger)
		childZLogger = logger.LoggerFromContext(childCtx)
		childZLogger.Info("Hello world 2")

	}

	zlogger := logger.LoggerFromContext(ctx)
	zlogger.Info("Hello world 3")
}
