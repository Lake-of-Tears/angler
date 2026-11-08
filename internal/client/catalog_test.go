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
