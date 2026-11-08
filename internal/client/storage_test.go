package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Lake-of-Tears/angler/internal/config"
)

func TestStorageClientEndpoints(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/storage/browse" && r.Method == http.MethodGet:
			prefix := r.URL.Query().Get("prefix")
			if prefix != "gold/telemetry" {
				t.Errorf("expected prefix gold/telemetry, got %s", prefix)
			}
			json.NewEncoder(w).Encode(StorageBrowseResponse{
				Bucket:       "lakehouse",
				Prefix:       "gold/telemetry",
				TotalFolders: 1,
				TotalFiles:   1,
				Folders: []StorageFolder{
					{Name: "year=2026", Prefix: "gold/telemetry/year=2026/"},
				},
				Files: []StorageFile{
					{
						Key:           "gold/telemetry/data.parquet",
						Name:          "data.parquet",
						Size:          1048576,
						SizeFormatted: "1.0 MB",
						IsParquet:     true,
					},
				},
			})

		case r.URL.Path == "/api/storage/objects/gold/telemetry/data.parquet/metadata" && r.Method == http.MethodGet:
			rowCount := int64(50000)
			colCount := 12
			json.NewEncoder(w).Encode(StorageMetadataResponse{
				Key:           "gold/telemetry/data.parquet",
				Size:          1048576,
				SizeFormatted: "1.0 MB",
				ContentType:   "application/octet-stream",
				ETag:          "etag-12345",
				IsParquet:     true,
				RowCount:      &rowCount,
				ColumnCount:   &colCount,
				Compression:   "SNAPPY",
			})

		case r.URL.Path == "/api/storage/objects/gold/telemetry/data.parquet/stream" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("mock stream content for data.parquet"))

		case r.URL.Path == "/api/storage/upload" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(StorageUploadResponse{
				Key:    "bronze/events.json",
				Size:   128,
				ETag:   "etag-upload-999",
				Status: "uploaded",
			})

		case r.URL.Path == "/api/storage/objects/bronze/events.json" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"deleted": true})

		case r.URL.Path == "/api/storage/stats" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(StorageStatsResponse{
				Bucket:       "lakehouse",
				TotalBytes:   100000000,
				TotalObjects: 42,
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c := NewClient(config.Context{Endpoint: ts.URL, Workspace: "default"})

	// 1. Test BrowseStorage
	browse, err := c.BrowseStorage("gold/telemetry", 50, "")
	if err != nil {
		t.Fatalf("BrowseStorage failed: %v", err)
	}
	if len(browse.Files) != 1 || browse.Files[0].Name != "data.parquet" {
		t.Errorf("unexpected browse files: %+v", browse.Files)
	}
	if len(browse.Folders) != 1 || browse.Folders[0].Name != "year=2026" {
		t.Errorf("unexpected browse folders: %+v", browse.Folders)
	}

	// 2. Test GetStorageMetadata
	meta, err := c.GetStorageMetadata("gold/telemetry/data.parquet")
	if err != nil {
		t.Fatalf("GetStorageMetadata failed: %v", err)
	}
	if !meta.IsParquet || *meta.RowCount != 50000 {
		t.Errorf("unexpected metadata: %+v", meta)
	}

	// 3. Test StreamObject & DownloadObject
	var buf bytes.Buffer
	if err := c.DownloadObject("gold/telemetry/data.parquet", &buf); err != nil {
		t.Fatalf("DownloadObject failed: %v", err)
	}
	if !strings.Contains(buf.String(), "mock stream content") {
		t.Errorf("unexpected downloaded stream content: %s", buf.String())
	}

	// 4. Test UploadObject
	uploadReader := strings.NewReader(`{"event": "login"}`)
	upResp, err := c.UploadObject("bronze", "events.json", uploadReader, "application/json")
	if err != nil {
		t.Fatalf("UploadObject failed: %v", err)
	}
	if upResp.Key != "bronze/events.json" {
		t.Errorf("unexpected upload key: %s", upResp.Key)
	}

	// 5. Test DeleteObject
	if err := c.DeleteObject("bronze/events.json", false); err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}

	// 6. Test GetStorageStats
	stats, err := c.GetStorageStats()
	if err != nil {
		t.Fatalf("GetStorageStats failed: %v", err)
	}
	if stats.TotalObjects != 42 {
		t.Errorf("expected 42 objects, got %d", stats.TotalObjects)
	}
}

func TestCleanStoragePath(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"lot://gold/users", "gold/users"},
		{"s3://lakehouse/bronze/events.parquet", "lakehouse/bronze/events.parquet"},
		{"dbfs:/volumes/main/default", "dbfs:/volumes/main/default"},
		{"dbfs://volumes/main/default", "volumes/main/default"},
		{"/silver/sales.parquet", "silver/sales.parquet"},
		{"silver/sales.parquet", "silver/sales.parquet"},
	}

	for _, tc := range cases {
		actual := CleanStoragePath(tc.input)
		if actual != tc.expected {
			t.Errorf("CleanStoragePath(%q) = %q, expected %q", tc.input, actual, tc.expected)
		}
	}
}
