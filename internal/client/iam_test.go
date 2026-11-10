package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestIAMAndAdminClientMethods(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// Users
		case r.URL.Path == "/api/users" && r.Method == http.MethodGet:
			dispName := "Lakehouse Admin"
			_ = json.NewEncoder(w).Encode([]User{
				{
					ID:          "u-1",
					Email:       "admin@lot.io",
					DisplayName: &dispName,
					Role:        "superadmin",
					IsActive:    true,
					CreatedAt:   now,
				},
			})

		case r.URL.Path == "/api/users/u-1" && r.Method == http.MethodPatch:
			var req UpdateUserRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			role := "user"
			if req.Role != nil {
				role = *req.Role
			}
			_ = json.NewEncoder(w).Encode(User{
				ID:        "u-1",
				Email:     "admin@lot.io",
				Role:      role,
				IsActive:  true,
				CreatedAt: now,
			})

		// Workspaces
		case r.URL.Path == "/api/workspaces" && r.Method == http.MethodGet:
			desc := "Primary analytics workspace"
			role := "admin"
			_ = json.NewEncoder(w).Encode([]Workspace{
				{
					ID:          "ws-1",
					Name:        "Analytics Prod",
					Slug:        "analytics-prod",
					Description: &desc,
					Status:      "active",
					MyRole:      &role,
					CreatedAt:   now,
				},
			})

		case r.URL.Path == "/api/workspaces/ws-1" && r.Method == http.MethodGet:
			desc := "Primary analytics workspace"
			role := "admin"
			_ = json.NewEncoder(w).Encode(Workspace{
				ID:          "ws-1",
				Name:        "Analytics Prod",
				Slug:        "analytics-prod",
				Description: &desc,
				Status:      "active",
				MyRole:      &role,
				CreatedAt:   now,
			})

		case r.URL.Path == "/api/workspaces" && r.Method == http.MethodPost:
			var req CreateWorkspaceRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			w.WriteHeader(http.StatusCreated)
			desc := req.Description
			role := "admin"
			_ = json.NewEncoder(w).Encode(Workspace{
				ID:          "ws-new",
				Name:        req.Name,
				Slug:        req.Slug,
				Description: &desc,
				Status:      "active",
				MyRole:      &role,
				CreatedAt:   now,
			})

		case r.URL.Path == "/api/workspaces/ws-1/members" && r.Method == http.MethodGet:
			disp := "Engineer"
			_ = json.NewEncoder(w).Encode([]WorkspaceMember{
				{
					UserID:      "u-2",
					Email:       "dev@lot.io",
					DisplayName: &disp,
					Role:        "editor",
					GlobalRole:  "user",
					JoinedAt:    now,
				},
			})

		case r.URL.Path == "/api/workspaces/ws-1/members" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})

		case r.URL.Path == "/api/workspaces/ws-1/members/u-2" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})

		// Admin settings
		case r.URL.Path == "/api/admin/settings" && r.Method == http.MethodGet:
			model := "gemini-2.0-flash"
			_ = json.NewEncoder(w).Encode(SystemSettings{
				CatalogSoftDeleteDays:           7,
				SessionInactivityTimeoutMinutes: 60,
				SessionMaxHours:                 24,
				GeminiAPIKeyConfigured:          true,
				GeminiDefaultModel:              &model,
			})

		case r.URL.Path == "/api/admin/settings" && r.Method == http.MethodPatch:
			var req UpdateSystemSettingsRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			model := "gemini-2.5-pro"
			if req.GeminiDefaultModel != nil {
				model = *req.GeminiDefaultModel
			}
			_ = json.NewEncoder(w).Encode(SystemSettings{
				CatalogSoftDeleteDays:           7,
				SessionInactivityTimeoutMinutes: 120,
				SessionMaxHours:                 24,
				GeminiAPIKeyConfigured:          true,
				GeminiDefaultModel:              &model,
			})

		// Admin sessions
		case r.URL.Path == "/api/admin/sessions" && r.Method == http.MethodGet:
			email := "admin@lot.io"
			ip := "192.168.1.10"
			_ = json.NewEncoder(w).Encode([]UserSession{
				{
					ID:           "sess-1",
					UserID:       "u-1",
					Email:        &email,
					IPAddress:    &ip,
					CreatedAt:    now,
					LastActiveAt: now,
					ExpiresAt:    now.Add(24 * time.Hour),
					IsRevoked:    false,
					IsCurrent:    true,
				},
			})

		case r.URL.Path == "/api/admin/sessions/sess-1" && r.Method == http.MethodDelete:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "revoked_id": "sess-1"})

		// Admin platform services
		case r.URL.Path == "/api/admin/services" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(PlatformServicesListResponse{
				ClusterConnected: true,
				Mode:             "in_cluster",
				Namespace:        "lake-of-tears",
				TotalServices:    1,
				HealthyCount:     1,
				Services: []PlatformServiceInfo{
					{ID: "superset", Name: "lake-of-tears-superset", DisplayName: "Apache Superset", Status: "running"},
				},
			})

		case r.URL.Path == "/api/admin/services/superset/restart" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(PlatformServiceActionResponse{
				Success: true, Service: "superset", Action: "restart",
			})

		case r.URL.Path == "/api/admin/services/superset/stop" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(PlatformServiceActionResponse{
				Success: true, Service: "superset", Action: "stop",
			})

		case r.URL.Path == "/api/admin/services/superset/start" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(PlatformServiceActionResponse{
				Success: true, Service: "superset", Action: "start",
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c := NewClient(config.Context{
		Endpoint: ts.URL,
		Token:    "test-tok",
	})

	// 1. Users
	users, err := c.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers error: %v", err)
	}
	if len(users) != 1 || users[0].Email != "admin@lot.io" {
		t.Errorf("unexpected users: %+v", users)
	}

	user, err := c.GetUser("u-1")
	if err != nil {
		t.Fatalf("GetUser error: %v", err)
	}
	if user.ID != "u-1" {
		t.Errorf("unexpected user: %+v", user)
	}

	newRole := "user"
	upUser, err := c.UpdateUser("u-1", UpdateUserRequest{Role: &newRole})
	if err != nil {
		t.Fatalf("UpdateUser error: %v", err)
	}
	if upUser.Role != "user" {
		t.Errorf("unexpected updated role: %s", upUser.Role)
	}

	// 2. Workspaces
	wsList, err := c.ListAllWorkspaces()
	if err != nil {
		t.Fatalf("ListAllWorkspaces error: %v", err)
	}
	if len(wsList) != 1 || wsList[0].Slug != "analytics-prod" {
		t.Errorf("unexpected workspaces: %+v", wsList)
	}

	wsItem, err := c.GetWorkspace("ws-1")
	if err != nil {
		t.Fatalf("GetWorkspace error: %v", err)
	}
	if wsItem.Name != "Analytics Prod" {
		t.Errorf("unexpected workspace: %+v", wsItem)
	}

	createdWs, err := c.CreateWorkspace(CreateWorkspaceRequest{
		Name: "Data Science",
		Slug: "data-science",
	})
	if err != nil {
		t.Fatalf("CreateWorkspace error: %v", err)
	}
	if createdWs.ID != "ws-new" {
		t.Errorf("unexpected created workspace: %+v", createdWs)
	}

	members, err := c.ListWorkspaceMembers("ws-1")
	if err != nil {
		t.Fatalf("ListWorkspaceMembers error: %v", err)
	}
	if len(members) != 1 || members[0].Email != "dev@lot.io" {
		t.Errorf("unexpected members: %+v", members)
	}

	if err := c.AddWorkspaceMember("ws-1", AddWorkspaceMemberRequest{Email: "new@lot.io", Role: "viewer"}); err != nil {
		t.Fatalf("AddWorkspaceMember error: %v", err)
	}

	if err := c.RemoveWorkspaceMember("ws-1", "u-2"); err != nil {
		t.Fatalf("RemoveWorkspaceMember error: %v", err)
	}

	// 3. Admin Settings & Sessions
	settings, err := c.GetSystemSettings()
	if err != nil {
		t.Fatalf("GetSystemSettings error: %v", err)
	}
	if settings.CatalogSoftDeleteDays != 7 {
		t.Errorf("unexpected settings: %+v", settings)
	}

	newTimeout := 120
	updatedSettings, err := c.UpdateSystemSettings(UpdateSystemSettingsRequest{SessionInactivityTimeoutMinutes: &newTimeout})
	if err != nil {
		t.Fatalf("UpdateSystemSettings error: %v", err)
	}
	if updatedSettings.SessionInactivityTimeoutMinutes != 120 {
		t.Errorf("unexpected updated settings: %+v", updatedSettings)
	}

	sessions, err := c.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	if len(sessions) != 1 || !sessions[0].IsCurrent {
		t.Errorf("unexpected sessions: %+v", sessions)
	}

	if err := c.RevokeSession("sess-1"); err != nil {
		t.Fatalf("RevokeSession error: %v", err)
	}

	// 4. Platform Services
	svcs, err := c.ListPlatformServices()
	if err != nil || len(svcs.Services) != 1 || svcs.Services[0].ID != "superset" {
		t.Fatalf("ListPlatformServices error: %v, %+v", err, svcs)
	}
	if _, err := c.RestartPlatformService("superset"); err != nil {
		t.Fatalf("RestartPlatformService error: %v", err)
	}
	if _, err := c.StopPlatformService("superset"); err != nil {
		t.Fatalf("StopPlatformService error: %v", err)
	}
	if _, err := c.StartPlatformService("superset"); err != nil {
		t.Fatalf("StartPlatformService error: %v", err)
	}
}
