package ui

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestFormatDataJSONYAMLTOON(t *testing.T) {
	sample := map[string]interface{}{
		"status":  "healthy",
		"version": "1.2.0",
	}

	jsonOut, err := FormatData(FormatJSON, sample)
	if err != nil {
		t.Fatalf("FormatData JSON failed: %v", err)
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(jsonOut), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if parsed["status"] != "healthy" {
		t.Errorf("expected healthy, got %s", parsed["status"])
	}

	yamlOut, err := FormatData(FormatYAML, sample)
	if err != nil {
		t.Fatalf("FormatData YAML failed: %v", err)
	}
	if !strings.Contains(yamlOut, "status: healthy") {
		t.Errorf("expected YAML to contain 'status: healthy', got:\n%s", yamlOut)
	}

	toonOut, err := FormatData(FormatTOON, sample)
	if err != nil {
		t.Fatalf("FormatData TOON failed: %v", err)
	}
	if !strings.Contains(toonOut, "status: healthy") || !strings.Contains(toonOut, "version: 1.2.0") {
		t.Errorf("expected TOON output to contain status and version, got:\n%s", toonOut)
	}
}

func TestMachineModeRoutingAndSuppression(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	OutWriter = &outBuf
	ErrWriter = &errBuf
	defer func() {
		OutWriter = os.Stdout
		ErrWriter = os.Stderr
	}()

	ttyTrue := true
	SetTTYOverrides(&ttyTrue, &ttyTrue)
	defer SetTTYOverrides(nil, nil)

	SetOutputFormat(FormatJSON)
	defer SetOutputFormat(FormatTable)

	// Banner must be suppressed in JSON mode
	PrintBanner()
	if outBuf.Len() != 0 {
		t.Errorf("expected banner to be suppressed on stdout in JSON mode, got %q", outBuf.String())
	}

	// Info and warnings must route to stderr so stdout remains clean
	PrintInfo("connecting to endpoint")
	PrintWarning("rate limit approaching")
	if outBuf.Len() != 0 {
		t.Errorf("expected stdout to remain empty during PrintInfo/PrintWarning, got %q", outBuf.String())
	}
	if !strings.Contains(errBuf.String(), "connecting to endpoint") || !strings.Contains(errBuf.String(), "rate limit approaching") {
		t.Errorf("expected info/warning on stderr, got %q", errBuf.String())
	}
}

func TestResolveExitCodeAndStructuredEmitError(t *testing.T) {
	cases := []struct {
		err  error
		code int
		kind string
	}{
		{nil, ExitOK, "ok"},
		{NewUsageError(errors.New("invalid argument")), ExitUsageError, "usage_error"},
		{errors.New("login failed (HTTP 401): invalid token"), ExitAuthError, "auth_error"},
		{errors.New("catalog error (HTTP 404): table not found"), ExitNotFoundError, "not_found"},
		{errors.New("failed to reach server at http://localhost:8000: connection refused"), ExitConnectionError, "connection_error"},
		{errors.New("unexpected internal failure"), ExitGeneralError, "runtime_error"},
	}

	for _, tc := range cases {
		got := ResolveExitCode(tc.err)
		if got != tc.code {
			t.Errorf("ResolveExitCode(%v) = %d, want %d", tc.err, got, tc.code)
		}
		if ErrorTypeName(got) != tc.kind {
			t.Errorf("ErrorTypeName(%d) = %s, want %s", got, ErrorTypeName(got), tc.kind)
		}
	}

	var errBuf bytes.Buffer
	ErrWriter = &errBuf
	defer func() {
		ErrWriter = os.Stderr
	}()
	SetOutputFormat(FormatJSON)
	defer SetOutputFormat(FormatTable)

	code := EmitError(NewAuthError(errors.New("unauthorized access")))
	if code != ExitAuthError {
		t.Fatalf("expected exit code %d, got %d", ExitAuthError, code)
	}

	var structured StructuredError
	if err := json.Unmarshal(errBuf.Bytes(), &structured); err != nil {
		t.Fatalf("expected valid JSON error on stderr, got %q: %v", errBuf.String(), err)
	}
	if structured.Code != ExitAuthError || structured.Type != "auth_error" {
		t.Errorf("unexpected structured error payload: %+v", structured)
	}
}
