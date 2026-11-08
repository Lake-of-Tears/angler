package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/client"
	"github.com/Lake-of-Tears/angler/internal/ui"
)

func resetIAMFlags() {
	userRole = ""
	userDisplayName = ""
	userActive = true
	wsAdminName = ""
	wsAdminSlug = ""
	wsAdminDescription = ""
	wsMemberEmail = ""
	wsMemberRole = "viewer"
	wsMemberForce = false
	adminSessionTimeout = 60
	adminMaxHours = 24
	adminGeminiModel = ""
	adminGeminiKey = ""
	adminForce = false
	jsonOutput = false
	outputFlag = ""
}

func TestIAMCommandsRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	var foundUsers, foundWorkspaces, foundAdmin bool
	for _, c := range commands {
		switch c.Name() {
		case "users":
			foundUsers = true
			subs := map[string]bool{"list": false, "get": false, "update": false}
			for _, sub := range c.Commands() {
				if _, ok := subs[sub.Name()]; ok {
					subs[sub.Name()] = true
				}
			}
			for s, found := range subs {
				if !found {
					t.Errorf("expected subcommand %q under users", s)
				}
			}

		case "workspaces":
			foundWorkspaces = true
			subs := map[string]bool{"list": false, "get": false, "create": false, "members": false}
			for _, sub := range c.Commands() {
				if _, ok := subs[sub.Name()]; ok {
					subs[sub.Name()] = true
				}
				if sub.Name() == "members" {
					mSubs := map[string]bool{"list": false, "add": false, "remove": false}
					for _, ms := range sub.Commands() {
						if _, ok := mSubs[ms.Name()]; ok {
							mSubs[ms.Name()] = true
						}
					}
					for msName, mFound := range mSubs {
						if !mFound {
							t.Errorf("expected subcommand %q under workspaces members", msName)
						}
					}
				}
			}
			for s, found := range subs {
				if !found {
					t.Errorf("expected subcommand %q under workspaces", s)
				}
			}

		case "admin":
			foundAdmin = true
			subs := map[string]bool{"settings": false, "sessions": false}
			for _, sub := range c.Commands() {
				if _, ok := subs[sub.Name()]; ok {
					subs[sub.Name()] = true
				}
			}
			for s, found := range subs {
				if !found {
					t.Errorf("expected subcommand %q under admin", s)
				}
			}
		}
	}

	if !foundUsers {
		t.Error("users command not registered on rootCmd")
	}
	if !foundWorkspaces {
		t.Error("workspaces command not registered on rootCmd")
	}
	if !foundAdmin {
		t.Error("admin command not registered on rootCmd")
	}
}

func setupIAMMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	now := time.Now().UTC().Truncate(time.Second)

	// Users endpoints
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		disp := "Admin User"
		users := []client.User{
			{
				ID:          "u-1",
				Email:       "admin@lot.io",
				DisplayName: &disp,
				Role:        "superadmin",
				IsActive:    true,
				CreatedAt:   now,
			},
		}
		_ = json.NewEncoder(w).Encode(users)
	})

	mux.HandleFunc("/api/users/u-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req client.UpdateUserRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		role := "user"
		if req.Role != nil {
			role = *req.Role
		}
		disp := "Updated Admin"
		_ = json.NewEncoder(w).Encode(client.User{
			ID:          "u-1",
			Email:       "admin@lot.io",
			DisplayName: &disp,
			Role:        role,
			IsActive:    true,
			CreatedAt:   now,
		})
	})

	// Workspaces endpoints
	mux.HandleFunc("/api/workspaces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			role := "admin"
			desc := "Analytics Core"
			_ = json.NewEncoder(w).Encode([]client.Workspace{
				{
					ID:          "ws-1",
					Name:        "Analytics",
					Slug:        "analytics",
					Description: &desc,
					Status:      "active",
					MyRole:      &role,
					CreatedAt:   now,
				},
			})
		case http.MethodPost:
			var req client.CreateWorkspaceRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			w.WriteHeader(http.StatusCreated)
			desc := req.Description
			_ = json.NewEncoder(w).Encode(client.Workspace{
				ID:          "ws-new",
				Name:        req.Name,
				Slug:        req.Slug,
				Description: &desc,
				Status:      "active",
				CreatedAt:   now,
			})
		}
	})

	mux.HandleFunc("/api/workspaces/ws-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		role := "admin"
		desc := "Analytics Core"
		_ = json.NewEncoder(w).Encode(client.Workspace{
			ID:          "ws-1",
			Name:        "Analytics",
			Slug:        "analytics",
			Description: &desc,
			Status:      "active",
			MyRole:      &role,
			CreatedAt:   now,
		})
	})

	mux.HandleFunc("/api/workspaces/ws-1/members", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			disp := "Analyst"
			_ = json.NewEncoder(w).Encode([]client.WorkspaceMember{
				{
					UserID:      "u-2",
					Email:       "analyst@lot.io",
					DisplayName: &disp,
					Role:        "editor",
					GlobalRole:  "user",
					JoinedAt:    now,
				},
			})
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	})

	mux.HandleFunc("/api/workspaces/ws-1/members/u-2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	// Admin Settings & Sessions
	mux.HandleFunc("/api/admin/settings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		model := "gemini-2.0-flash"
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(client.SystemSettings{
				CatalogSoftDeleteDays:            7,
				WorkspaceInactiveGracePeriodDays: 30,
				SessionInactivityTimeoutMinutes:  60,
				SessionMaxHours:                  24,
				GeminiAPIKeyConfigured:           true,
				GeminiDefaultModel:               &model,
			})
		case http.MethodPatch:
			var req client.UpdateSystemSettingsRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			timeout := 120
			if req.SessionInactivityTimeoutMinutes != nil {
				timeout = *req.SessionInactivityTimeoutMinutes
			}
			_ = json.NewEncoder(w).Encode(client.SystemSettings{
				CatalogSoftDeleteDays:            7,
				WorkspaceInactiveGracePeriodDays: 30,
				SessionInactivityTimeoutMinutes:  timeout,
				SessionMaxHours:                  24,
				GeminiAPIKeyConfigured:           true,
				GeminiDefaultModel:               &model,
			})
		}
	})

	mux.HandleFunc("/api/admin/sessions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		email := "admin@lot.io"
		ip := "127.0.0.1"
		_ = json.NewEncoder(w).Encode([]client.UserSession{
			{
				ID:           "sess-123",
				UserID:       "u-1",
				Email:        &email,
				IPAddress:    &ip,
				CreatedAt:    now,
				LastActiveAt: now,
				ExpiresAt:    now.Add(24 * time.Hour),
				IsCurrent:    true,
			},
		})
	})

	mux.HandleFunc("/api/admin/sessions/sess-123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "revoked_id": "sess-123"})
	})

	return httptest.NewServer(mux)
}

func TestUsersCLI(t *testing.T) {
	ts := setupIAMMockServer(t)
	defer ts.Close()

	// 1. Users list table
	resetIAMFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "users", "list", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "admin@lot.io") {
		t.Errorf("expected email in stdout: %s", stdout)
	}

	// 2. Users list JSON
	resetIAMFlags()
	stdout, stderr, code = executeCLIForTest(t, false, "user", "ls", "--json", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	var users []client.User
	if err := json.Unmarshal([]byte(stdout), &users); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if len(users) != 1 || users[0].Role != "superadmin" {
		t.Errorf("unexpected users JSON: %+v", users)
	}

	// 3. Users get
	resetIAMFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "users", "get", "u-1", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "User: admin@lot.io") {
		t.Errorf("expected user details in stdout: %s", stdout)
	}

	// 4. Users update
	resetIAMFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "users", "update", "u-1", "--role", "user", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "updated successfully") {
		t.Errorf("expected update success message: %s", stdout)
	}
}

func TestWorkspacesCLI(t *testing.T) {
	ts := setupIAMMockServer(t)
	defer ts.Close()

	// 1. Workspaces list
	resetIAMFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "workspaces", "list", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Analytics") {
		t.Errorf("expected Analytics in stdout: %s", stdout)
	}

	// 2. Workspaces get
	resetIAMFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "workspaces", "get", "ws-1", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Workspace: Analytics") {
		t.Errorf("expected workspace get details: %s", stdout)
	}

	// 3. Workspaces create
	resetIAMFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "workspaces", "create", "--name", "Growth Team", "--slug", "growth", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "created successfully") {
		t.Errorf("expected creation success message: %s", stdout)
	}

	// 4. Workspaces members list
	resetIAMFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "workspaces", "members", "list", "ws-1", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "analyst@lot.io") {
		t.Errorf("expected member in stdout: %s", stdout)
	}

	// 5. Workspaces members add
	resetIAMFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "workspaces", "members", "add", "ws-1", "--email", "newbie@lot.io", "--role", "viewer", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "added to workspace") {
		t.Errorf("expected add success message: %s", stdout)
	}

	// 6. Workspaces members remove
	resetIAMFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "workspaces", "members", "remove", "ws-1", "u-2", "--force", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "removed from workspace") {
		t.Errorf("expected remove message: %s", stdout)
	}
}

func TestAdminCLI(t *testing.T) {
	ts := setupIAMMockServer(t)
	defer ts.Close()

	// 1. Admin settings get
	resetIAMFlags()
	stdout, stderr, code := executeCLIForTest(t, true, "admin", "settings", "get", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Catalog Soft-Delete Days:       7") {
		t.Errorf("expected soft-delete days in stdout: %s", stdout)
	}

	// 2. Admin settings update
	resetIAMFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "admin", "settings", "update", "--session-timeout", "120", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "System settings updated successfully") {
		t.Errorf("expected settings updated message: %s", stdout)
	}

	// 3. Admin sessions list
	resetIAMFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "admin", "sessions", "list", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "sess-123") {
		t.Errorf("expected session id: %s", stdout)
	}

	// 4. Admin sessions revoke
	resetIAMFlags()
	stdout, stderr, code = executeCLIForTest(t, true, "admin", "sessions", "revoke", "sess-123", "--force", "--endpoint", ts.URL, "--token", "test-tok")
	if code != ui.ExitOK {
		t.Fatalf("expected code 0, got %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "revoked successfully") {
		t.Errorf("expected session revoke message: %s", stdout)
	}
}
