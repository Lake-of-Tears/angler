package toon

import (
	"strings"
	"testing"
)

func TestMarshalAndUnmarshalTable(t *testing.T) {
	type Item struct {
		Name   string `json:"name"`
		Format string `json:"format"`
		Rows   int    `json:"rows"`
	}

	input := map[string]interface{}{
		"project": "lake-of-tears",
		"enabled": true,
		"datasets": []Item{
			{Name: "telemetry", Format: "parquet", Rows: 1500},
			{Name: "events", Format: "iceberg", Rows: 4200},
		},
	}

	raw, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	out := string(raw)
	if !strings.Contains(out, "datasets[2]{format,name,rows}:") {
		t.Fatalf("expected TOON tabular header 'datasets[2]{format,name,rows}:', got:\n%s", out)
	}
	if !strings.Contains(out, "  parquet,telemetry,1500") {
		t.Fatalf("expected first TOON row, got:\n%s", out)
	}

	var decoded struct {
		Project  string `json:"project"`
		Enabled  bool   `json:"enabled"`
		Datasets []Item `json:"datasets"`
	}
	if err := Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Project != "lake-of-tears" || !decoded.Enabled {
		t.Errorf("unexpected scalar fields: %+v", decoded)
	}
	if len(decoded.Datasets) != 2 || decoded.Datasets[0].Name != "telemetry" || decoded.Datasets[1].Rows != 4200 {
		t.Errorf("unexpected datasets after roundtrip: %+v", decoded.Datasets)
	}
}

func TestMarshalQueryResponseColumnsAndRows(t *testing.T) {
	queryResp := map[string]interface{}{
		"columns":   []string{"id", "region"},
		"rows":      [][]interface{}{{1, "us-east"}, {2, "eu-west"}},
		"row_count": 2,
		"duration":  "8ms",
	}

	raw, err := Marshal(queryResp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	out := string(raw)
	if !strings.Contains(out, "rows[2]{id,region}:") {
		t.Fatalf("expected converted TOON table 'rows[2]{id,region}:', got:\n%s", out)
	}
	if !strings.Contains(out, "  1,us-east") {
		t.Fatalf("expected converted row '1,us-east', got:\n%s", out)
	}
}
