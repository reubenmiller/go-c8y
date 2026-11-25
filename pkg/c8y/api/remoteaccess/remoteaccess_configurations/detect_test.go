package remoteaccess_configurations

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// detectServer serves the given configurations for device d1, and the one
// with id "7" by id.
func detectServer(t *testing.T, configs string) (*Service, func()) {
	t.Helper()
	return testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/service/remoteaccess/devices/d1/configurations":
			_, _ = w.Write([]byte(configs))
		case "/service/remoteaccess/devices/d1/configurations/7":
			_, _ = w.Write([]byte(`{"id":"7","name":"by-id","protocol":"PASSTHROUGH","port":8080}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	})
}

func TestDetect(t *testing.T) {
	const configs = `[
		{"id":"1","name":"webssh","protocol":"SSH","port":22},
		{"id":"2","name":"http:web","protocol":"PASSTHROUGH","port":80},
		{"id":"3","name":"native-ssh","protocol":"PASSTHROUGH","port":2222},
		{"id":"4","name":"Router","protocol":"PASSTHROUGH","port":22},
		{"id":"42","name":"42","protocol":"PASSTHROUGH","port":443}
	]`
	tests := []struct {
		name, ref, wantID, wantErr string
	}{
		{"by id", "7", "7", ""},
		{"id-like name", "42", "42", ""},
		{"exact name, any case", "router", "4", ""},
		{"wildcard", "http:*", "2", ""},
		{"only passthrough", "webssh", "", "not found: name=webssh"},
		{"no match", "nope*", "", "not found: name=nope*"},
		{"automatic: port 22 first", "", "4", ""},
	}
	svc, closeFn := detectServer(t, configs)
	defer closeFn()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := svc.Detect(context.Background(), DetectOptions{ManagedObjectID: "d1", Name: tc.ref})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Detect: %v", err)
			}
			if cfg.ID() != tc.wantID {
				t.Errorf("id = %q, want %q", cfg.ID(), tc.wantID)
			}
		})
	}
}

func TestDetectAutomatic(t *testing.T) {
	tests := []struct {
		name, configs, wantID, wantErr string
	}{
		{"none", `[{"id":"1","name":"vnc","protocol":"VNC","port":5900}]`, "", "no remote access passthrough configuration"},
		{"the only one", `[{"id":"1","name":"web","protocol":"PASSTHROUGH","port":80}]`, "1", ""},
		{"ssh in the name", `[{"id":"1","name":"web","protocol":"PASSTHROUGH","port":80},{"id":"2","name":"My-SSH","protocol":"PASSTHROUGH","port":2222}]`, "2", ""},
		{"else the first", `[{"id":"1","name":"web","protocol":"PASSTHROUGH","port":80},{"id":"2","name":"api","protocol":"PASSTHROUGH","port":8080}]`, "1", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, closeFn := detectServer(t, tc.configs)
			defer closeFn()
			cfg, err := svc.Detect(context.Background(), DetectOptions{ManagedObjectID: "d1"})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Detect: %v", err)
			}
			if cfg.ID() != tc.wantID {
				t.Errorf("id = %q, want %q", cfg.ID(), tc.wantID)
			}
		})
	}
}
