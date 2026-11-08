package ui

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Lake-of-Tears/angler/internal/toon"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"gopkg.in/yaml.v3"
)

// CLIError wraps an error with a deterministic process exit code.
type CLIError struct {
	Err  error
	Code int
}

func (e *CLIError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *CLIError) Unwrap() error {
	return e.Err
}

func NewCLIError(code int, err error) error {
	if err == nil {
		return nil
	}
	return &CLIError{Err: err, Code: code}
}

func NewUsageError(err error) error {
	return NewCLIError(ExitUsageError, err)
}

func NewAuthError(err error) error {
	return NewCLIError(ExitAuthError, err)
}

func NewNotFoundError(err error) error {
	return NewCLIError(ExitNotFoundError, err)
}

func NewConnectionError(err error) error {
	return NewCLIError(ExitConnectionError, err)
}

// StructuredError is the machine-readable error payload written to stderr.
type StructuredError struct {
	Error string `json:"error" yaml:"error"`
	Code  int    `json:"code" yaml:"code"`
	Type  string `json:"type" yaml:"type"`
}

// ResolveExitCode inspects an error and returns a deterministic exit code.
func ResolveExitCode(err error) int {
	if err == nil {
		return ExitOK
	}

	var cliErr *CLIError
	if errors.As(err, &cliErr) && cliErr.Code != 0 {
		return cliErr.Code
	}

	msg := strings.ToLower(err.Error())

	// Check for HTTP status or semantic indicators
	switch {
	case strings.Contains(msg, "http 401") ||
		strings.Contains(msg, "http 403") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "authentication failed") ||
		strings.Contains(msg, "login failed"):
		return ExitAuthError
	case strings.Contains(msg, "http 404") ||
		strings.Contains(msg, "not found"):
		return ExitNotFoundError
	case strings.Contains(msg, "failed to reach server") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "context deadline exceeded"):
		return ExitConnectionError
	case strings.Contains(msg, "unknown flag") ||
		strings.Contains(msg, "unknown shorthand flag") ||
		strings.Contains(msg, "unknown command") ||
		strings.Contains(msg, "accepts ") ||
		strings.Contains(msg, "requires at least") ||
		strings.Contains(msg, "invalid output format") ||
		strings.Contains(msg, "invalid schema format") ||
		strings.Contains(msg, "no sql query provided") ||
		strings.Contains(msg, "please provide a question") ||
		strings.Contains(msg, "http 400") ||
		strings.Contains(msg, "http 422") ||
		strings.Contains(msg, "syntax error"):
		return ExitUsageError
	default:
		return ExitGeneralError
	}
}

// ErrorTypeName returns a stable symbolic identifier for an exit code.
func ErrorTypeName(code int) string {
	switch code {
	case ExitOK:
		return "ok"
	case ExitUsageError:
		return "usage_error"
	case ExitAuthError:
		return "auth_error"
	case ExitNotFoundError:
		return "not_found"
	case ExitConnectionError:
		return "connection_error"
	default:
		return "runtime_error"
	}
}

// ValidateFormat checks whether the requested output format is supported.
func ValidateFormat(format string, allowTabular bool) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case FormatTable, FormatJSON, FormatYAML, FormatTOON:
		return nil
	case FormatCSV, FormatNDJSON:
		if allowTabular {
			return nil
		}
		return NewUsageError(fmt.Errorf("invalid output format %q (supported: table, json, yaml, toon)", format))
	default:
		if allowTabular {
			return NewUsageError(fmt.Errorf("invalid output format %q (supported: table, json, yaml, toon, csv, ndjson)", format))
		}
		return NewUsageError(fmt.Errorf("invalid output format %q (supported: table, json, yaml, toon)", format))
	}
}

// FormatData serializes data into json, yaml, or toon string representation.
func FormatData(format string, data interface{}) (string, error) {
	switch strings.ToLower(format) {
	case FormatJSON:
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return string(b), nil
	case FormatYAML:
		// Normalize via JSON tags first so struct field names are consistent across JSON, YAML, and TOON.
		normalized, err := normalizeViaJSON(data)
		if err != nil {
			return "", fmt.Errorf("failed to normalize data for YAML: %w", err)
		}
		b, err := yaml.Marshal(normalized)
		if err != nil {
			return "", fmt.Errorf("failed to marshal YAML: %w", err)
		}
		return strings.TrimRight(string(b), "\n"), nil
	case FormatTOON:
		b, err := toon.Marshal(data)
		if err != nil {
			return "", fmt.Errorf("failed to marshal TOON: %w", err)
		}
		return strings.TrimRight(string(b), "\n"), nil
	default:
		return "", fmt.Errorf("unsupported structured format %q", format)
	}
}

func normalizeViaJSON(data interface{}) (interface{}, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var out interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RenderOutput writes data in the active structured format (json, yaml, toon) or invokes humanFn for table mode.
func RenderOutput(data interface{}, humanFn func()) error {
	format := GetOutputFormat()
	switch format {
	case FormatJSON, FormatYAML, FormatTOON:
		rendered, err := FormatData(format, data)
		if err != nil {
			return err
		}
		fmt.Fprintln(OutWriter, rendered)
		return nil
	default:
		if humanFn != nil {
			humanFn()
		}
		return nil
	}
}

// RenderTable builds a Lipgloss table (styled on TTY, plain border without ANSI colors on non-TTY).
func RenderTable(headers []string, rows [][]string) string {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		Headers(headers...).
		Rows(rows...)

	if !IsMachineOutput() {
		t = t.BorderStyle(lipgloss.NewStyle().Foreground(BorderColor))
	}

	return t.String()
}

// RenderCSV writes headers and rows in RFC 4180 CSV format to OutWriter.
func RenderCSV(headers []string, rows [][]string) error {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if len(headers) > 0 {
		if err := writer.Write(headers); err != nil {
			return err
		}
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	_, err := fmt.Fprint(OutWriter, buf.String())
	return err
}

// RenderNDJSON writes rows as newline-delimited JSON objects to OutWriter.
func RenderNDJSON(headers []string, rows [][]interface{}) error {
	for _, row := range rows {
		record := make(map[string]interface{}, len(headers))
		for i, h := range headers {
			if i < len(row) {
				record[h] = row[i]
			} else {
				record[h] = nil
			}
		}
		data, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("failed to encode NDJSON record: %w", err)
		}
		if _, err := fmt.Fprintln(OutWriter, string(data)); err != nil {
			return err
		}
	}
	return nil
}

// RenderNDJSONObjects writes objects as newline-delimited JSON objects to OutWriter.
func RenderNDJSONObjects(records []map[string]interface{}) error {
	for _, record := range records {
		data, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("failed to encode NDJSON record: %w", err)
		}
		if _, err := fmt.Fprintln(OutWriter, string(data)); err != nil {
			return err
		}
	}
	return nil
}

// EmitError writes the error to ErrWriter (structured in json/yaml/toon mode, styled text in table mode)
// and returns the resolved exit code.
func EmitError(err error) int {
	if err == nil {
		return ExitOK
	}

	code := ResolveExitCode(err)
	format := GetOutputFormat()

	switch format {
	case FormatJSON, FormatYAML, FormatTOON:
		payload := StructuredError{
			Error: err.Error(),
			Code:  code,
			Type:  ErrorTypeName(code),
		}
		if out, mErr := FormatData(format, payload); mErr == nil {
			fmt.Fprintln(ErrWriter, out)
			return code
		}
	}

	PrintError("%v", err)
	return code
}
