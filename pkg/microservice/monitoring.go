package microservice

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// AddHealthEndpointHandlers adds a set of health endpoints to the microservice
// The following endpoints are added:
//   - /prometheus
//   - /health
//   - /env
//   - /logfile
func (m *Microservice) AddHealthEndpointHandlers(e *echo.Echo) {
	if e == nil {
		zap.S().Errorf("Failed to end health endpoitns because the echo server is nil")
		return
	}

	zap.S().Infof("Adding /prometheus, /health, /env and /logfile endpoints to the microservice")

	e.GET("/prometheus", echo.WrapHandler(promhttp.Handler()))
	e.GET("/health", m.HealthHandler)
	e.GET("/env", m.EnvironmentVariablesHandler)
	e.GET("/logfile", m.GetLogFileHandler)
}

// EnvironmentVariablesHandler return the list of environment variables
func (m *Microservice) EnvironmentVariablesHandler(c echo.Context) error {
	// System settings
	systemProperties := map[string]interface{}{}
	for _, key := range m.Config.AllKeys() {
		value := m.Config.GetString(key)
		if strings.Contains(key, "password") {
			systemProperties[key] = "*******************************************"
		} else {
			systemProperties[key] = value
		}
	}

	// Environment variables
	environmentVariables := map[string]interface{}{}
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if strings.Contains(strings.ToLower(pair[0]), "password") {
			environmentVariables[pair[0]] = "*******************************************"
		} else {
			environmentVariables[pair[0]] = pair[1]
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"systemEnvironment": environmentVariables,
		"systemProperties":  systemProperties,
	})
}

// GetLogFileHandler get the current log file in json format
func (m *Microservice) GetLogFileHandler(c echo.Context) error {
	filepath := m.Config.GetString("log.file")

	text := ""
	tail := c.QueryParam("tail")
	lines, err := strconv.ParseInt(tail, 10, 64)

	if err == nil {
		if lines > 200 {
			lines = 200
		}
		text = getLastLineWithSeek(filepath, int64(lines))
	} else {
		b, err := os.ReadFile(filepath)

		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"reason": fmt.Sprintf("Could not read log file: %s", err),
			})
		}

		text = string(b)
	}

	return c.String(http.StatusOK, text)
}

// HealthHandler returns health endpoint
func (m *Microservice) HealthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"status": "UP",
	})
}

func getLastLineWithSeek(filepath string, numberLines int64) string {
	fileHandle, err := os.Open(filepath)

	if err != nil {
		panic("Cannot open file")
	}
	defer fileHandle.Close()

	line := ""
	var cursor int64
	stat, _ := fileHandle.Stat()
	filesize := stat.Size()
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

		if cursor == -filesize { // stop if we are at the beginning
			break
		}
	}

	return line
}
