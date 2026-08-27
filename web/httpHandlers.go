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

func filesPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caddyConfig := readCaddyfile()
		units := readUnitFiles()
		webNodes := readWebTree()
		if err := FilesPage(caddyConfig, units, webNodes).Render(r.Context(), w); err != nil {
			slog.Debug("render error", "component", "FilesPage", "err", err)
		}
	}
}

func quadletsPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := QuadletsPage(runningQuadletServices()).Render(r.Context(), w); err != nil {
			slog.Debug("render error", "component", "EtcPage", "err", err)
		}
	}
}

// logLines caps how many journal lines are fetched per tail.
const logLines = 50

// quadletLogsSSE streams a periodic journalctl tail for one Quadlet service,
// picked by the ?service= query param. The name is validated against the
// currently running services rather than trusted, since it's client input
// and gets passed straight to exec.Command.
func quadletLogsSSE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service := r.URL.Query().Get("service")
		if !isRunningQuadletService(service) {
			http.Error(w, "unknown or not-running service", http.StatusBadRequest)
			return
		}

		sse := datastar.NewSSE(w, r, datastar.WithCompression(datastar.WithBrotli()))

		ticker := time.NewTicker(time.Duration(UpdateTick) * (2 * time.Second))
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				status := readServiceStatus(service)
				logs := readServiceLogs(service, logLines)
				if err := sse.PatchElementTempl(ServiceLogPane(service, status, logs)); err != nil {
					return
				}
			}
		}
	}
}
