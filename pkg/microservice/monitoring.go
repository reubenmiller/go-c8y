package microservice

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RegisterHealthEndpoints adds the standard microservice endpoints to the given
// mux:
//
//   - /prometheus
//   - /health
//   - /env
//   - /logfile
//
// The handlers are plain net/http handlers so they can also be mounted
// individually on any router (e.g. chi, gorilla, or echo via echo.WrapHandler).
func (m *Microservice) RegisterHealthEndpoints(mux *http.ServeMux) {
	slog.Info("Adding /prometheus, /health, /env and /logfile endpoints to the microservice")

	mux.Handle("/prometheus", promhttp.Handler())
	mux.HandleFunc("/health", m.HealthHandler)
	mux.HandleFunc("/env", m.EnvironmentVariablesHandler)
	mux.HandleFunc("/logfile", m.GetLogFileHandler)
}

// PrometheusHandler returns the Prometheus metrics handler served on /prometheus.
func (m *Microservice) PrometheusHandler() http.Handler {
	return promhttp.Handler()
}

// HealthHandler returns the health endpoint
func (m *Microservice) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "UP",
	})
}

// EnvironmentVariablesHandler returns the list of environment variables and
// configuration properties. Values of keys containing "password" are masked.
func (m *Microservice) EnvironmentVariablesHandler(w http.ResponseWriter, _ *http.Request) {
	const masked = "*******************************************"

	// System settings
	systemProperties := map[string]any{}
	for _, key := range m.Config.AllKeys() {
		if strings.Contains(key, "password") {
			systemProperties[key] = masked
		} else {
			systemProperties[key] = m.Config.GetString(key)
		}
	}

	// Environment variables
	environmentVariables := map[string]any{}
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) != 2 {
			continue
		}
		if strings.Contains(strings.ToLower(pair[0]), "password") {
			environmentVariables[pair[0]] = masked
		} else {
			environmentVariables[pair[0]] = pair[1]
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"systemEnvironment": environmentVariables,
		"systemProperties":  systemProperties,
	})
}

// GetLogFileHandler returns the current log file contents. The optional "tail"
// query parameter limits the output to the last n lines (capped at 200).
func (m *Microservice) GetLogFileHandler(w http.ResponseWriter, r *http.Request) {
	filepath := m.Config.GetString("log.file")

	text := ""
	tail := r.URL.Query().Get("tail")
	lines, err := strconv.ParseInt(tail, 10, 64)

	if err == nil {
		if lines > 200 {
			lines = 200
		}
		text, err = getLastLineWithSeek(filepath, lines)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"reason": fmt.Sprintf("Could not read log file: %s", err),
			})
			return
		}
	} else {
		b, err := os.ReadFile(filepath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"reason": fmt.Sprintf("Could not read log file: %s", err),
			})
			return
		}
		text = string(b)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, text)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Warn("Failed to encode response", "err", err)
	}
}

func getLastLineWithSeek(filepath string, numberLines int64) (string, error) {
	fileHandle, err := os.Open(filepath)

	if err != nil {
		return "", err
	}
	defer fileHandle.Close()

	line := ""
	var cursor int64
	stat, err := fileHandle.Stat()
	if err != nil {
		return "", err
	}
	fileSize := stat.Size()
	var totalLines int64
	for {
		cursor--
		fileHandle.Seek(cursor, io.SeekEnd)

		char := make([]byte, 1)
		fileHandle.Read(char)

		if cursor != -1 && (char[0] == 10 || char[0] == 13) { // stop if we find a line
			totalLines++
		}

		if totalLines >= numberLines {
			break
		}

		line = fmt.Sprintf("%s%s", string(char), line) // there is more efficient way

		if cursor == -fileSize { // stop if we are at the beginning
			break
		}
	}

	return line, nil
}
