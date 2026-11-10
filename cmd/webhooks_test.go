package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/ui"
)

func TestWebhooksAndJobsWebhookCommands(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/webhooks/default/secret", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workspace_id":   "ws-1",
			"workspace_slug": "default",
			"webhook_secret": "whsec_test_12345",
			"webhook_url":    "/api/v1/ingest/webhooks/default",
		})
	})

	mux.HandleFunc("/api/v1/webhooks/default/secret/regenerate", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workspace_id":   "ws-1",
			"workspace_slug": "default",
			"webhook_secret": "whsec_rotated_999",
			"webhook_url":    "/api/v1/ingest/webhooks/default",
		})
	})

	mux.HandleFunc("/api/v1/webhooks/default", func(w http.ResponseWriter, r *http.Request) {
		sig := r.Header.Get("X-Hub-Signature-256")
		if sig == "" || !strings.HasPrefix(sig, "sha256=") {
			t.Fatalf("expected HMAC signature header X-Hub-Signature-256, got %q", sig)
		}
		trig := r.URL.Query().Get("trigger_pipeline")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":             "accepted",
			"event_id":           "evt-101",
			"workspace_slug":     "default",
			"received_at":        "2026-11-10T12:00:00Z",
			"triggered_pipeline": trig,
			"dag_run_id":         "webhook__20261110T120000",
		})
	})

	mux.HandleFunc("/api/v1/pipelines/telemetry_etl/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"dag_id":                 "telemetry_etl",
				"color_hex":              "#8b5cf6",
				"display_label":          "Telemetry ETL",
				"is_visible_on_timeline": true,
				"webhook_enabled":        body["webhook_enabled"],
				"webhook_url":            "/api/v1/pipelines/default/telemetry_etl/webhook",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"dag_id":                 "telemetry_etl",
			"color_hex":              "#8b5cf6",
			"display_label":          "Telemetry ETL",
			"is_visible_on_timeline": true,
			"webhook_enabled":        true,
			"webhook_url":            "/api/v1/pipelines/default/telemetry_etl/webhook",
		})
	})

	mux.HandleFunc("/api/v1/pipelines/default/telemetry_etl/webhook", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":         "queued",
			"dag_id":         "telemetry_etl",
			"dag_run_id":     "webhook_run_777",
			"state":          "queued",
			"workspace_slug": "default",
			"event_id":       "evt-pipe-1",
		})
	})

	mux.HandleFunc("/api/v1/pipelines/timeline", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workspace": "default",
			"date":      "2026-11-10",
			"timezone":  "UTC",
			"pipelines": []map[string]any{
				{"dag_id": "telemetry_etl", "display_label": "Telemetry ETL", "schedule_interval": "@hourly", "color_hex": "#8b5cf6"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	var outBuf, errBuf bytes.Buffer
	ui.OutWriter = &outBuf
	ui.ErrWriter = &errBuf
	t.Cleanup(func() {
		ui.OutWriter = os.Stdout
		ui.ErrWriter = os.Stderr
	})

	// 1. webhooks secret get
	outBuf.Reset()
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "table", "webhooks", "secret", "get"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("webhooks secret get failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "whsec_test_12345") {
		t.Fatalf("expected webhook secret in output, got: %s", outBuf.String())
	}

	// 2. webhooks send with --hmac and --trigger-pipeline
	outBuf.Reset()
	rootCmd.SetArgs([]string{
		"--endpoint", srv.URL,
		"--output", "table",
		"webhooks", "send",
		"--payload", `{"source":"sensor","val":42}`,
		"--trigger-pipeline", "telemetry_etl",
		"--hmac",
	})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("webhooks send failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "evt-101") || !strings.Contains(outBuf.String(), "telemetry_etl") {
		t.Fatalf("expected event ID and triggered pipeline in output, got: %s", outBuf.String())
	}

	// 3. jobs webhook enable
	outBuf.Reset()
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "table", "jobs", "webhook", "enable", "telemetry_etl"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("jobs webhook enable failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "/api/v1/pipelines/default/telemetry_etl/webhook") {
		t.Fatalf("expected webhook URL in output, got: %s", outBuf.String())
	}

	// 4. jobs webhook trigger
	outBuf.Reset()
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "table", "jobs", "webhook", "trigger", "telemetry_etl", "--payload", `{"batch":1}`})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("jobs webhook trigger failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "webhook_run_777") {
		t.Fatalf("expected webhook_run_777 in output, got: %s", outBuf.String())
	}

	// 5. jobs timeline
	outBuf.Reset()
	rootCmd.SetArgs([]string{"--endpoint", srv.URL, "--output", "table", "jobs", "timeline", "--date", "2026-11-10"})
	if code := ExecuteRoot(); code != ui.ExitOK {
		t.Fatalf("jobs timeline failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "Telemetry ETL") {
		t.Fatalf("expected Telemetry ETL in timeline output, got: %s", outBuf.String())
	}
}
