// Command centos-streamed is a tiny Cockpit-style system monitor: a single Go
// binary that serves a live web view of the machine it runs on — host facts and
// the running process list — pushed to the browser over server-sent events.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	streamed "github.com/json-bateman/centos-streamed"
	"github.com/json-bateman/centos-streamed/web"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	// CTRL+C sends SIGINT; `kill` and service stop send SIGTERM. Both trigger a graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	streamed.LoadSettings()

	if err := web.RunBlocking(ctx); err != nil {
		return fmt.Errorf("run web server: %w", err)
	}
	return nil
}
