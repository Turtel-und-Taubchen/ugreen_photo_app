package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"photocurator/internal/api"
	"photocurator/internal/config"
	"photocurator/internal/probe"
	"photocurator/internal/version"
	"photocurator/web"
)

func main() {
	args := os.Args[1:]
	cmd := "serve"
	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "serve":
		os.Exit(serve(args))
	case "probe":
		os.Exit(probe.Run(args, os.Stderr))
	case "healthcheck":
		os.Exit(healthcheck(os.Getenv("PC_LISTEN")))
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\nusage: photocurator [serve|probe|healthcheck]\n", cmd)
		os.Exit(2)
	}
}

func serve(args []string) int {
	fl := flag.NewFlagSet("serve", flag.ContinueOnError)
	if err := fl.Parse(args); err != nil {
		return 2
	}
	addr := os.Getenv("PC_LISTEN")
	if addr == "" {
		addr = ":8080"
	}
	if err := checkStart(config.PathsFromEnv(os.Getenv)); err != nil {
		slog.Error("invalid configuration, refusing to start", "err", err)
		return 1
	}
	static, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		slog.Error("embedded frontend", "err", err)
		return 1
	}
	srv := &http.Server{Addr: addr, Handler: api.NewHandler(static), ReadHeaderTimeout: 10 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	slog.Info("photocurator listening", "addr", addr, "version", version.Version)

	select {
	case err := <-errc:
		slog.Error("server", "err", err)
		return 1
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("shutdown", "err", err)
		return 1
	}
	return 0
}

// checkStart must pass before any port is opened.
func checkStart(p config.Paths) error {
	return config.ValidatePaths(p)
}
