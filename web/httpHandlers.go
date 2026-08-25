package web

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

var UpdateTick = 1

type TickRequest struct {
	SelectValue string `json:"select_value"`
}

func homePage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := Home(collectServerInfo()).Render(r.Context(), w); err != nil {
			slog.Debug("render error", "component", "Page", "err", err)
		}
	}
}

func homePageSse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r, datastar.WithCompression(datastar.WithBrotli()))

		ticker := time.NewTicker(time.Duration(UpdateTick) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				if err := sse.PatchElementTempl(Home(collectServerInfo())); err != nil {
					return
				}
			}
		}
	}
}

// processLimit caps how many processes the table shows (top N by memory).
const processLimit = 40

func processesPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := ProcessesPage(collectProcesses(processLimit)).Render(r.Context(), w); err != nil {
			slog.Debug("render error", "component", "Page", "err", err)
		}
	}
}

func processesPageSse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r, datastar.WithCompression(datastar.WithBrotli()))

		ticker := time.NewTicker(time.Duration(UpdateTick) * (2 * time.Second))
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				if err := sse.PatchElementTempl(ProcessesPage(collectProcesses(processLimit))); err != nil {
					return
				}
			}
		}
	}
}

func etcPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caddyConfig := readCaddyfile()
		files := readUnitFiles()
		if err := EtcPage(caddyConfig, files).Render(r.Context(), w); err != nil {
			slog.Debug("render error", "component", "EtcPage", "err", err)
		}
	}
}

func etcPageSSE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r, datastar.WithCompression(datastar.WithBrotli()))

		ticker := time.NewTicker(time.Duration(UpdateTick) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				caddyConfig := readCaddyfile()
				containers := readUnitFiles()
				if err := sse.PatchElementTempl(EtcPage(caddyConfig, containers)); err != nil {
					return
				}
			}
		}
	}
}

func sshPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := SshPage().Render(r.Context(), w); err != nil {
			slog.Debug("render error", "component", "EtcPage", "err", err)
		}
	}
}

func varWwwPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := VarWwwPage(readWebTree()).Render(r.Context(), w); err != nil {
			slog.Debug("render error", "component", "StaticPage", "err", err)
		}
	}
}
