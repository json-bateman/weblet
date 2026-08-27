package web

import (
	"context"
	"embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"

	"github.com/benbjohnson/hashfs"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/json-bateman/weblet"
)

//go:embed static/*
var StaticFS embed.FS

// StaticSys serves embedded static files under content-hashed names so they can
// be cached forever; when a file changes its hash changes and busts the cache.
var StaticSys = hashfs.NewFS(StaticFS)

var CommitHash = "dev"

const (
	HomeUrl      = "/"
	ProcessesUrl = "/proc"
	FilesUrl     = "/files"
	QuadletsUrl  = "/quadlets"
)

// StaticPath returns the hashed URL for a file under static/, e.g.
// StaticPath("css/main.css") -> "/static/css/main.abc123.css".
func StaticPath(format string, args ...any) string {
	return "/" + StaticSys.HashName(fmt.Sprintf("static/"+format, args...))
}

// cacheUnhashedStatic stamps an ETag on static assets requested by a plain,
// unhashed path (e.g. a file pulled in via a CSS @import or a JS dynamic
// import, which never goes through StaticPath) so http.ServeContent - called
// inside hashfs's own handler - can answer future requests with a 304
// instead of re-sending the whole file on every navigation.
func cacheUnhashedStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if _, hash := hashfs.ParseName(name); hash == "" {
			if _, h := hashfs.ParseName(StaticSys.HashName(name)); h != "" {
				w.Header().Set("ETag", "\""+h+"\"")
				w.Header().Set("Cache-Control", "public, max-age=3600, must-revalidate")
			}
		}
		next.ServeHTTP(w, r)
	})
}

func getCommitHash() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func setupRoutes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get(HomeUrl, homePage())
	r.Get(HomeUrl+"sse", homePageSse())

	r.Get(ProcessesUrl, processesPage())
	r.Get(ProcessesUrl+"/sse", processesPageSse())

	r.Get(FilesUrl, filesPage())
	r.Get(FilesUrl+"/sse", filesPageSSE())

	r.Get(QuadletsUrl, sshPage())

	// Serve files embedded in the binary.
	r.Handle("/static/*", cacheUnhashedStatic(hashfs.FileServer(StaticSys)))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if err := NotFound().Render(r.Context(), w); err != nil {
			slog.Debug("render error", "component", "NotFound", "err", err)
		}
	})

	return r
}

// RunBlocking starts the HTTP server and blocks until setupCtx is cancelled, at
// which point it shuts down gracefully.
func RunBlocking(setupCtx context.Context) error {
	if CommitHash == "dev" {
		CommitHash = getCommitHash()
	}
	router := setupRoutes()

	addr := fmt.Sprintf(":%d", weblet.Env.Port)
	srv := http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		<-setupCtx.Done()
		log.Printf("shutdown 💽__💽")
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
	}()

	log.Printf("Starting server on http://localhost%s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Error starting server: %v", err)
	}
	return nil
}
