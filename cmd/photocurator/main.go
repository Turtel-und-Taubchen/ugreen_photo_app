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
	_ "time/tzdata" // the runtime image has no zoneinfo, and TZ must resolve

	"photocurator/internal/api"
	"photocurator/internal/auth"
	"photocurator/internal/config"
	"photocurator/internal/exif"
	"photocurator/internal/library"
	"photocurator/internal/metadata"
	"photocurator/internal/probe"
	"photocurator/internal/store"
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
	case "hash-password":
		os.Exit(hashPassword(os.Stdin, os.Stdout, os.Stderr))
	case "healthcheck":
		os.Exit(healthcheck(os.Getenv("PC_LISTEN")))
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\nusage: photocurator [serve|probe|hash-password|healthcheck]\n", cmd)
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
	paths := config.PathsFromEnv(os.Getenv)
	users, scan, err := checkStart(paths, os.Getenv)
	if err != nil {
		slog.Error("invalid configuration, refusing to start", "err", err)
		return 1
	}
	db, err := store.OpenAppDB(paths.DataDir)
	if err != nil {
		slog.Error("app database", "err", err)
		return 1
	}
	defer db.Close()
	static, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		slog.Error("embedded frontend", "err", err)
		return 1
	}
	idxDB, err := store.OpenIndexDB(paths.DataDir)
	if err != nil {
		slog.Error("index database", "err", err)
		return 1
	}
	defer idxDB.Close()
	exifConfig, err := metadata.WriteConfig(os.TempDir())
	if err != nil {
		slog.Error("exiftool config", "err", err)
		return 1
	}
	scanner := &library.Scanner{
		Root:   paths.PhotosDir,
		DB:     idxDB,
		Reader: exif.Reader{ConfigPath: exifConfig, Tags: metadata.ReadTags},
		Quiet:  scan.Quiet,
		Now:    time.Now,
		Loc:    time.Local,
	}
	sched := library.NewScheduler(scanner.Run, scan.Interval, time.After)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	// Registered after idxDB.Close, so the scan loop has ended before the index closes.
	scanCtx, cancelScan := context.WithCancel(ctx)
	scanDone := make(chan struct{})
	go func() { defer close(scanDone); sched.Run(scanCtx) }()
	defer func() { cancelScan(); <-scanDone }()

	srv := &http.Server{Addr: addr, Handler: api.NewHandler(static, api.AuthDeps{
		Users:    users,
		Sessions: auth.NewSessionStore(db, nil),
		Limiter:  auth.NewLoginLimiter(nil),
	}, api.LibraryDeps{Scan: library.Control{Scheduler: sched, DB: idxDB}}), ReadHeaderTimeout: 10 * time.Second}
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

// checkStart (paths, users file, scan settings) must pass before any port is opened.
func checkStart(p config.Paths, get func(string) string) (auth.Users, config.Scan, error) {
	if err := config.ValidatePaths(p); err != nil {
		return nil, config.Scan{}, err
	}
	users, err := auth.LoadUsers(p.UsersFile)
	if err != nil {
		return nil, config.Scan{}, err
	}
	scan, err := config.ScanFromEnv(get)
	return users, scan, err
}
