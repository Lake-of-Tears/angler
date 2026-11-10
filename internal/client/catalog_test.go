package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestCatalogClientEndpoints(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/catalogs" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]Catalog{
				{
					ID:        "cat-001",
					Name:      "main",
					Slug:      "main",
					IsDefault: true,
					CreatedAt: time.Now(),
				},
			})
		case r.URL.Path == "/api/catalogs/cat-001" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(Catalog{
				ID:        "cat-001",
				Name:      "main",
				Slug:      "main",
				IsDefault: true,
			})
		case r.URL.Path == "/api/catalogs" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(Catalog{
				ID:   "cat-002",
				Name: "analytics",
				Slug: "analytics",
			})
		case r.URL.Path == "/api/catalogs/cat-002" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/api/catalogs/cat-001/schemas" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]CatalogSchema{
				{
					ID:        "sch-001",
					CatalogID: "cat-001",
					Name:      "default",
					Slug:      "default",
					Tier:      "bronze",
				},
			})
		case r.URL.Path == "/api/catalogs/cat-001/schemas" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(CatalogSchema{
				ID:        "sch-002",
				CatalogID: "cat-001",
				Name:      "gold",
				Slug:      "gold",
			})
		case r.URL.Path == "/api/catalogs/cat-001/schemas/sch-001/tables" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]CatalogTable{
				{
					ID:       "tbl-001",
					SchemaID: "sch-001",
					Name:     "users",
					Slug:     "users",
				},
			})
		case r.URL.Path == "/api/catalogs/cat-001/schemas/sch-001/tables/tbl-001" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(CatalogTable{
				ID:          "tbl-001",
				SchemaID:    "sch-001",
				Name:        "users",
				Slug:        "users",
				Description: "Users dimension table",
			})
		case r.URL.Path == "/api/catalogs/manifest" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"catalog": "main", "schemas": []string{"default", "gold"}},
			})
		case r.URL.Path == "/api/workspaces/default/anomalies/detect" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(AnomalyDetectResponse{
				OK:        true,
				Workspace: "default",
				Summary:   map[string]interface{}{"anomalies_detected": 0},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c := NewClient(config.Context{Endpoint: ts.URL, Workspace: "default"})

	// 1. ListCatalogs
	catalogs, err := c.ListCatalogs()
	if err != nil || len(catalogs) != 1 || catalogs[0].Name != "main" {
		t.Fatalf("ListCatalogs failed: %v, %+v", err, catalogs)
	}

	// 2. GetCatalog
	cat, err := c.GetCatalog("cat-001")
	if err != nil || cat.ID != "cat-001" {
		t.Fatalf("GetCatalog failed: %v", err)
	}

	// 3. CreateCatalog
	newCat, err := c.CreateCatalog(CreateCatalogRequest{Name: "analytics"})
	if err != nil || newCat.ID != "cat-002" {
		t.Fatalf("CreateCatalog failed: %v", err)
	}

	// 4. DeleteCatalog
	if err := c.DeleteCatalog("cat-002", false); err != nil {
		t.Fatalf("DeleteCatalog failed: %v", err)
	}

	// 5. ListSchemas
	schemas, err := c.ListSchemas("cat-001")
	if err != nil || len(schemas) != 1 || schemas[0].Name != "default" {
		t.Fatalf("ListSchemas failed: %v", err)
	}

	// 6. ListTables & GetTable
	tables, err := c.ListTables("cat-001", "sch-001")
	if err != nil || len(tables) != 1 || tables[0].Name != "users" {
		t.Fatalf("ListTables failed: %v", err)
	}
	tbl, err := c.GetTable("cat-001", "sch-001", "tbl-001")
	if err != nil || tbl.Description != "Users dimension table" {
		t.Fatalf("GetTable failed: %v", err)
	}

	// 7. GetCatalogManifest
	manifest, err := c.GetCatalogManifest()
	if err != nil || len(manifest) != 1 {
		t.Fatalf("GetCatalogManifest failed: %v", err)
	}

	// 8. DetectAnomalies
	anomalyResp, err := c.DetectAnomalies(AnomalyDetectRequest{})
	if err != nil || !anomalyResp.OK {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}
}

func TestCatalogTrashTOONUpdateAndAccessClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/catalogs/trash" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]DeletedCatalogAsset{
				{ID: "tr-1", Name: "old_tbl", AssetType: "table", DaysRemaining: 5.5},
			})
		case r.URL.Path == "/api/catalogs/trash/tr-1/restore" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "restored_id": "tr-1"})
		case r.URL.Path == "/api/catalogs/trash/tr-1/purge" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "purged_id": "tr-1"})
		case r.URL.Path == "/api/catalogs/trash/purge-expired" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "purged_count": 3})
		case r.URL.Path == "/api/catalogs/cat-001" && r.Method == http.MethodPatch:
			json.NewEncoder(w).Encode(Catalog{ID: "cat-001", Name: "main", Description: "Updated"})
		case r.URL.Path == "/api/catalogs/cat-001/schemas/sch-001" && r.Method == http.MethodPatch:
			json.NewEncoder(w).Encode(CatalogSchema{ID: "sch-001", Name: "default", Description: "Updated"})
		case r.URL.Path == "/api/catalogs/cat-001/schemas/sch-001/tables/tbl-001" && r.Method == http.MethodPatch:
			json.NewEncoder(w).Encode(CatalogTable{ID: "tbl-001", Name: "users", Description: "Updated"})
		case r.URL.Path == "/api/catalogs/cat-001/schemas/sch-001/toon" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(TOONExportResponse{OK: true, TOON: "schema: default"})
		case r.URL.Path == "/api/catalogs/cat-001/schemas/sch-001/tables/tbl-001/toon" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(TOONExportResponse{OK: true, TOON: "table: users"})
		case r.URL.Path == "/api/catalogs/directory" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]CatalogDirectoryItem{
				{ID: "cat-ext", Name: "external_cat", AccessStatus: "approved"},
			})
		case r.URL.Path == "/api/catalogs/cat-ext/access" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]CatalogAccessItem{
				{ID: "acc-1", CatalogID: "cat-ext", Mode: "read", Status: "pending"},
			})
		case r.URL.Path == "/api/catalogs/cat-ext/access/request" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		case r.URL.Path == "/api/catalogs/cat-ext/access/acc-1" && r.Method == http.MethodPatch:
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		case r.URL.Path == "/api/catalogs/cat-ext/access/acc-1" && r.Method == http.MethodDelete:
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c := NewClient(config.Context{Endpoint: ts.URL, Workspace: "default", WorkspaceID: "ws-1"})

	// Trash
	trash, err := c.ListCatalogTrash()
	if err != nil || len(trash) != 1 || trash[0].ID != "tr-1" {
		t.Fatalf("ListCatalogTrash failed: %v", err)
	}
	if _, err := c.RestoreCatalogTrashAsset("tr-1", nil); err != nil {
		t.Fatalf("RestoreCatalogTrashAsset failed: %v", err)
	}
	if _, err := c.PurgeCatalogTrashAsset("tr-1"); err != nil {
		t.Fatalf("PurgeCatalogTrashAsset failed: %v", err)
	}
	if _, err := c.PurgeExpiredCatalogTrash(); err != nil {
		t.Fatalf("PurgeExpiredCatalogTrash failed: %v", err)
	}

	// Update
	desc := "Updated"
	if _, err := c.UpdateCatalog("cat-001", UpdateCatalogRequest{Description: &desc}); err != nil {
		t.Fatalf("UpdateCatalog failed: %v", err)
	}
	if _, err := c.UpdateSchema("cat-001", "sch-001", UpdateSchemaRequest{Description: &desc}); err != nil {
		t.Fatalf("UpdateSchema failed: %v", err)
	}
	if _, err := c.UpdateTable("cat-001", "sch-001", "tbl-001", UpdateTableRequest{Description: &desc}); err != nil {
		t.Fatalf("UpdateTable failed: %v", err)
	}

	// TOON
	schToon, err := c.GetSchemaTOON("cat-001", "sch-001")
	if err != nil || schToon.TOON != "schema: default" {
		t.Fatalf("GetSchemaTOON failed: %v", err)
	}
	tblToon, err := c.GetTableTOON("cat-001", "sch-001", "tbl-001")
	if err != nil || tblToon.TOON != "table: users" {
		t.Fatalf("GetTableTOON failed: %v", err)
	}

	// Directory & Access
	dir, err := c.ListCatalogDirectory("ws-1")
	if err != nil || len(dir) != 1 || dir[0].Name != "external_cat" {
		t.Fatalf("ListCatalogDirectory failed: %v", err)
	}
	acc, err := c.ListCatalogAccess("cat-ext")
	if err != nil || len(acc) != 1 || acc[0].ID != "acc-1" {
		t.Fatalf("ListCatalogAccess failed: %v", err)
	}
	if _, err := c.RequestCatalogAccess("cat-ext", RequestCatalogAccessRequest{Message: "Test"}); err != nil {
		t.Fatalf("RequestCatalogAccess failed: %v", err)
	}
	if _, err := c.UpdateCatalogAccess("cat-ext", "acc-1", "approved"); err != nil {
		t.Fatalf("UpdateCatalogAccess failed: %v", err)
	}
	if err := c.RevokeCatalogAccess("cat-ext", "acc-1"); err != nil {
		t.Fatalf("RevokeCatalogAccess failed: %v", err)
	}
}
