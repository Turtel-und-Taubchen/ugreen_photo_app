package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
	"photocurator/internal/thumbs"
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
	thumbDir := filepath.Join(paths.DataDir, "thumbs")
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		slog.Error("thumbnail cache", "err", err)
		return 1
	}
	thumbSvc := thumbs.NewService(thumbDir, paths.PhotosDir, thumbs.Renderer{CacheDir: thumbDir, Root: paths.PhotosDir}, 1)
	// Every completed scan queues the small previews of the whole index in browse order.
	scanAndQueue := func(ctx context.Context, progress func(done, total int)) (library.Result, error) {
		res, err := scanner.Run(ctx, progress)
		if err != nil {
			return res, err
		}
		all, lerr := library.AllPhotos(idxDB)
		if lerr != nil {
			slog.Error("thumbnail backlog", "err", lerr)
			return res, nil
		}
		entries := make([]thumbs.Entry, len(all))
		for i, p := range all {
			entries[i] = thumbEntry(p)
		}
		thumbSvc.EnqueueBackground(entries)
		return res, nil
	}
	sched := library.NewScheduler(scanAndQueue, scan.Interval, time.After)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	// Registered after idxDB.Close, so the scan loop has ended before the index closes.
	scanCtx, cancelScan := context.WithCancel(ctx)
	scanDone := make(chan struct{})
	go func() { defer close(scanDone); sched.Run(scanCtx) }()
	defer func() { cancelScan(); <-scanDone }()
	thumbSvc.Start(ctx)
	defer thumbSvc.Stop()

	srv := &http.Server{Addr: addr, Handler: api.NewHandler(static, api.AuthDeps{
		Users:    users,
		Sessions: auth.NewSessionStore(db, nil),
		Limiter:  auth.NewLoginLimiter(nil),
	}, api.LibraryDeps{
		Scan: library.Control{Scheduler: sched, DB: idxDB},
		Thumbs: &api.ThumbDeps{
			Lookup: func(id int64) (thumbs.Entry, bool, error) {
				p, ok, err := library.PhotoByID(idxDB, id)
				return thumbEntry(p), ok, err
			},
			Service:  thumbSvc,
			CacheDir: thumbDir,
		},
		Photos: photoList{idxDB},
		Index:  idxDB,
		Tags:   &api.TagDeps{App: db, Index: idxDB},
	}), ReadHeaderTimeout: 10 * time.Second}
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

func thumbEntry(p library.Photo) thumbs.Entry {
	return thumbs.Entry{ID: p.ID, RelPath: p.RelPath, MediaType: p.MediaType, Size: p.Size, MtimeNs: p.MtimeNs}
}

type photoList struct{ db *sql.DB }

func (l photoList) List(ctx context.Context, month string, after *library.ListCursor, limit int) ([]library.Photo, error) {
	if month != "" {
		return library.ListMonth(ctx, l.db, month, after, limit)
	}
	return library.ListPhotos(l.db, after, limit)
}

func (l photoList) Groups(ctx context.Context) (library.Groups, error) {
	return library.GroupCounts(ctx, l.db)
}

func (l photoList) Detail(ctx context.Context, id int64) (library.Detail, bool, error) {
	return library.PhotoDetail(ctx, l.db, id)
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
