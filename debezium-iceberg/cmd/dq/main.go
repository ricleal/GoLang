// Command dq runs the data-quality loop (`dq run`, default) or the Iceberg
// maintenance job (`dq maintain`) of the CDC stand.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"debezium-iceberg/internal/dq"
)

func main() {
	cmd := "run"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	cfg := dq.LoadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch cmd {
	case "run":
		if err := dq.Run(ctx, cfg); err != nil && ctx.Err() == nil {
			log.Fatalf("dq run: %v", err)
		}
	case "maintain":
		if err := dq.Maintain(ctx, cfg); err != nil {
			log.Fatalf("dq maintain: %v", err)
		}
	default:
		log.Fatalf("unknown command %q (want run|maintain)", cmd)
	}
}
