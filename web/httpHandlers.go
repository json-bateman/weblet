package web

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

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

		ticker := time.NewTicker(time.Second)
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

		ticker := time.NewTicker(2 * time.Second)
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

// quadletLogsSSE streams periodic journalctl tails for every currently running Quadlet service
func quadletLogsSSE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r, datastar.WithCompression(datastar.WithBrotli()))

		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				for _, service := range runningQuadletServices() {
					status := readServiceStatus(service)
					logs := readServiceLogs(service, logLines)
					if err := sse.PatchElementTempl(ServiceLogPane(service, status, logs)); err != nil {
						return
					}
				}
			}
		}
	}
}
