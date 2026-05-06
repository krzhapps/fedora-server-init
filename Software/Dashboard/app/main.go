package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/krzhalovski/fedora-server-init/dashboard/internal/modules/containers"
	"github.com/krzhalovski/fedora-server-init/dashboard/internal/modules/files"
	"github.com/krzhalovski/fedora-server-init/dashboard/internal/modules/snippets"
	"github.com/krzhalovski/fedora-server-init/dashboard/internal/server"
)

//go:embed all:templates
var templatesFS embed.FS

//go:embed all:static
var staticFS embed.FS

func main() {
	addr := envOr("LISTEN_ADDR", ":8090")
	mediaRoot := envOr("MEDIA_ROOT", "/root/jellyfin/media")
	snippetsFile := envOr("SNIPPETS_FILE", "/root/snippets.json")

	templates, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		log.Fatalf("templates sub: %v", err)
	}
	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("static sub: %v", err)
	}

	srv, err := server.New(server.Config{
		Templates: templates,
		Static:    static,
		Modules: []server.Module{
			containers.New(),
			snippets.New(snippetsFile),
			files.New(mediaRoot),
		},
	})
	if err != nil {
		log.Fatalf("server init: %v", err)
	}

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("dashboard listening on %s (media root: %s)", addr, mediaRoot)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
