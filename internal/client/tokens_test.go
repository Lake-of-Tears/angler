package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestCurrentUser(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/me" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-pat-token" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"detail": "Invalid token"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           "11111111-2222-3333-4444-555555555555",
			"email":        "test@lakeoftears.io",
			"display_name": "Test User",
			"role":         "admin",
			"is_active":    true,
		})
	}))
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint: ts.URL,
		Token:    "test-pat-token",
	})

	user, err := c.CurrentUser()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if user.Email != "test@lakeoftears.io" {
		t.Errorf("expected email test@lakeoftears.io, got %s", user.Email)
	}
	if user.Role != "admin" {
		t.Errorf("expected role admin, got %s", user.Role)
	}
}

func TestListWorkspaces(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspaces" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":      "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				"name":    "Default Workspace",
				"slug":    "default",
				"status":  "active",
				"my_role": "admin",
			},
		})
	}))
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint: ts.URL,
		Token:    "some-token",
	})

	workspaces, err := c.ListWorkspaces()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}
	if workspaces[0].Slug != "default" {
		t.Errorf("expected slug default, got %s", workspaces[0].Slug)
	}
}

func TestCreateToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/tokens" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req TokenCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Name != "ci-bot" {
			t.Errorf("expected name ci-bot, got %s", req.Name)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           "99999999-8888-7777-6666-555555555555",
			"name":         req.Name,
			"token":        "lot_pat_abcdef0123456789abcdef",
			"token_prefix": "lot_pat_abcdef01",
			"created_at":   time.Now().Format(time.RFC3339),
		})
	}))
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint: ts.URL,
		Token:    "session-token",
	})

	days := 30
	res, err := c.CreateToken(&TokenCreateRequest{
		Name:         "ci-bot",
		LifetimeDays: &days,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Token != "lot_pat_abcdef0123456789abcdef" {
		t.Errorf("expected raw token, got %s", res.Token)
	}
}

func TestListAndRevokeTokens(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/auth/tokens" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":           "tok-1",
					"name":         "agent-token",
					"token_prefix": "lot_pat_1111",
					"is_revoked":   false,
					"created_at":   time.Now().Format(time.RFC3339),
				},
			})
		case r.URL.Path == "/api/auth/tokens/tok-1" && r.Method == http.MethodDelete:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":         true,
				"revoked_id": "tok-1",
			})
		default:
			t.Fatalf("unexpected call: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint: ts.URL,
		Token:    "session-token",
	})

	tokens, err := c.ListTokens(false)
	if err != nil {
		t.Fatalf("failed to list tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}

	rev, err := c.RevokeToken("tok-1")
	if err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}
	if !rev.OK || rev.RevokedID != "tok-1" {
		t.Errorf("unexpected revoke response: %+v", rev)
	}
}
