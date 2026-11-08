package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestAIClientMethods(t *testing.T) {
	mux := http.NewServeMux()

	// 1. /api/ai/chat
	mux.HandleFunc("/api/ai/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req GenieChatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		resp := GenieChatResponse{
			OK:      true,
			Mode:    req.Mode,
			Answer:  "There are 3 customer tables in your catalog.",
			SQL:     "SELECT * FROM gold.customers LIMIT 10",
			Columns: []string{"id", "name", "spend"},
			Data: []map[string]any{
				{"id": 1, "name": "Alice", "spend": 450.0},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 2. /api/ai/chat/stream
	mux.HandleFunc("/api/ai/chat/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		fmt.Fprintf(w, "event: step\ndata: {\"step\": \"planning\", \"message\": \"Formulating query\"}\n\n")
		flusher.Flush()

		resultJSON, _ := json.Marshal(GenieChatResponse{
			OK:     true,
			Answer: "Streamed answer complete.",
			SQL:    "SELECT 1",
		})
		fmt.Fprintf(w, "event: result\ndata: %s\n\n", string(resultJSON))
		flusher.Flush()
	})

	// 3. /api/ai/context
	mux.HandleFunc("/api/ai/context", func(w http.ResponseWriter, r *http.Request) {
		resp := AIContextResponse{
			OK:            true,
			WorkspaceID:   "ws-123",
			WorkspaceSlug: "lake-ws",
			WorkspaceName: "Lakehouse Dev",
			Role:          "editor",
			Catalogs: []map[string]any{
				{"name": "production_gold"},
			},
			DefaultDatasets: []map[string]any{
				{"name": "telemetry"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 4. /api/admin/settings/test-gemini-key
	mux.HandleFunc("/api/admin/settings/test-gemini-key", func(w http.ResponseWriter, r *http.Request) {
		resp := TestGeminiKeyResponse{
			OK:      true,
			Model:   "gemini-2.0-flash",
			Message: "Gemini API key is valid and authorized.",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// 5. /api/catalogs/cat-1/schemas/sch-1/tables/tbl-1/vector-index/sync
	mux.HandleFunc("/api/catalogs/cat-1/schemas/sch-1/tables/tbl-1/vector-index/sync", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "syncing"})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint:  ts.URL,
		Token:     "test-token",
		Workspace: "lake-ws",
	})

	// Test GenieChat
	chatResp, err := c.GenieChat(GenieChatRequest{
		Question: "Which tables exist?",
		Mode:     "auto",
	})
	if err != nil {
		t.Fatalf("GenieChat error: %v", err)
	}
	if !chatResp.OK || !strings.Contains(chatResp.Answer, "customer tables") {
		t.Errorf("unexpected chat response: %+v", chatResp)
	}

	// Test GenieChatStream
	var eventsReceived []string
	var finalAnswer string
	err = c.GenieChatStream(GenieChatRequest{Question: "Stream test"}, func(event string, raw []byte) error {
		eventsReceived = append(eventsReceived, event)
		if event == "result" {
			var res GenieChatResponse
			if err := json.Unmarshal(raw, &res); err == nil {
				finalAnswer = res.Answer
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("GenieChatStream error: %v", err)
	}
	if len(eventsReceived) != 2 || eventsReceived[0] != "step" || eventsReceived[1] != "result" {
		t.Errorf("unexpected streamed events: %v", eventsReceived)
	}
	if finalAnswer != "Streamed answer complete." {
		t.Errorf("unexpected final answer: %s", finalAnswer)
	}

	// Test GetAIContext
	ctxResp, err := c.GetAIContext()
	if err != nil {
		t.Fatalf("GetAIContext error: %v", err)
	}
	if ctxResp.WorkspaceSlug != "lake-ws" || len(ctxResp.Catalogs) != 1 {
		t.Errorf("unexpected context response: %+v", ctxResp)
	}

	// Test TestGeminiKey
	keyResp, err := c.TestGeminiKey("AIzaSyTestKey", "gemini-2.0-flash")
	if err != nil {
		t.Fatalf("TestGeminiKey error: %v", err)
	}
	if !keyResp.OK {
		t.Errorf("expected OK true, got false")
	}

	// Test SyncTableVectorIndex
	if err := c.SyncTableVectorIndex("cat-1", "sch-1", "tbl-1"); err != nil {
		t.Fatalf("SyncTableVectorIndex error: %v", err)
	}
}
