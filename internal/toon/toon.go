package toon

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	tableHeaderRegex = regexp.MustCompile(`^([\w-]+)\[(\d*)\]\{([^}]+)\}:\s*$`)
	intRegex         = regexp.MustCompile(`^-?\d+$`)
	floatRegex       = regexp.MustCompile(`^-?\d+\.\d+$`)
)

// Marshal serializes any Go value into Token-Oriented Object Notation (TOON).
func Marshal(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("null\n"), nil
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("toon marshal: failed to normalize value: %w", err)
	}

	var normalized interface{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&normalized); err != nil {
		return nil, fmt.Errorf("toon marshal: failed to decode normalized value: %w", err)
	}

	// If top-level value is a slice, wrap it in a root "items" key so it forms a valid TOON table/list.
	switch val := normalized.(type) {
	case []interface{}:
		normalized = map[string]interface{}{
			"items": val,
		}
	case map[string]interface{}:
		// Special handling for SQL QueryResponse-like structures with columns + rows ([][]interface{})
		if colsRaw, hasCols := val["columns"]; hasCols {
			if rowsRaw, hasRows := val["rows"]; hasRows {
				if converted, ok := convertColumnsAndRows(colsRaw, rowsRaw); ok {
					cloned := make(map[string]interface{}, len(val))
					for k, v2 := range val {
						if k == "columns" {
							continue
						}
						if k == "rows" {
							cloned["rows"] = converted
							continue
						}
						cloned[k] = v2
					}
					normalized = cloned
				}
			}
		}
	}

	if m, ok := normalized.(map[string]interface{}); ok {
		out := dumpMap(m, 0)
		return []byte(out), nil
	}

	return []byte(fmt.Sprintf("%v\n", formatScalar(normalized))), nil
}

func convertColumnsAndRows(colsRaw, rowsRaw interface{}) ([]interface{}, bool) {
	colSlice, ok := colsRaw.([]interface{})
	if !ok || len(colSlice) == 0 {
		return nil, false
	}
	rowSlice, ok := rowsRaw.([]interface{})
	if !ok || len(rowSlice) == 0 {
		return nil, false
	}

	cols := make([]string, len(colSlice))
	for i, c := range colSlice {
		cols[i] = fmt.Sprintf("%v", c)
	}

	result := make([]interface{}, 0, len(rowSlice))
	for _, r := range rowSlice {
		cells, ok := r.([]interface{})
		if !ok {
			return nil, false
		}
		rowObj := make(map[string]interface{}, len(cols))
		for i, colName := range cols {
			if i < len(cells) {
				rowObj[colName] = cells[i]
			} else {
				rowObj[colName] = nil
			}
		}
		result = append(result, rowObj)
	}
	return result, true
}

func dumpMap(data map[string]interface{}, indent int) string {
	var lines []string
	pad := strings.Repeat(" ", indent)

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := data[k]
		switch typed := v.(type) {
		case []interface{}:
			if len(typed) > 0 {
				if firstMap, ok := typed[0].(map[string]interface{}); ok && isUniformObjectSlice(typed) {
					cols := sortedKeys(firstMap)
					lines = append(lines, fmt.Sprintf("%s%s[%d]{%s}:", pad, k, len(typed), strings.Join(cols, ",")))
					for _, item := range typed {
						rowMap := item.(map[string]interface{})
						rowVals := make([]string, len(cols))
						for i, c := range cols {
							cell := rowMap[c]
							var valStr string
							if cell == nil {
								valStr = ""
							} else {
								valStr = formatScalar(cell)
							}
							if strings.ContainsAny(valStr, ",\"\n") {
								escaped := strings.ReplaceAll(valStr, `"`, `""`)
								valStr = fmt.Sprintf(`"%s"`, escaped)
							}
							rowVals[i] = valStr
						}
						lines = append(lines, fmt.Sprintf("%s  %s", pad, strings.Join(rowVals, ",")))
					}
					continue
				}
			}
			inner := make([]string, len(typed))
			for i, item := range typed {
				inner[i] = formatScalar(item)
			}
			lines = append(lines, fmt.Sprintf("%s%s: [%s]", pad, k, strings.Join(inner, ", ")))
		case map[string]interface{}:
			if len(typed) == 0 {
				lines = append(lines, fmt.Sprintf("%s%s: null", pad, k))
			} else {
				lines = append(lines, fmt.Sprintf("%s%s:", pad, k))
				sub := strings.TrimRight(dumpMap(typed, indent+2), "\n")
				lines = append(lines, sub)
			}
		case nil:
			lines = append(lines, fmt.Sprintf("%s%s: null", pad, k))
		default:
			lines = append(lines, fmt.Sprintf("%s%s: %s", pad, k, formatScalar(typed)))
		}
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func isUniformObjectSlice(items []interface{}) bool {
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok || len(m) == 0 {
			return false
		}
		for _, v := range m {
			switch v.(type) {
			case map[string]interface{}, []interface{}:
				return false
			}
		}
	}
	return true
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatScalar(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case bool:
		if val {
			return "true"
		}
		return "false"
	case json.Number:
		return val.String()
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// Unmarshal parses a TOON document into the target Go value.
func Unmarshal(data []byte, v interface{}) error {
	parsed := ParseMap(string(data))
	rawJSON, err := json.Marshal(parsed)
	if err != nil {
		return fmt.Errorf("toon unmarshal: %w", err)
	}
	return json.Unmarshal(rawJSON, v)
}

// ParseMap parses a TOON string into a map[string]interface{}.
func ParseMap(text string) map[string]interface{} {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	result := make(map[string]interface{})
	i := 0
	totalLines := len(lines)

	for i < totalLines {
		rawLine := lines[i]
		line := strings.TrimSpace(rawLine)

		if line == "" || strings.HasPrefix(line, "#") {
			i++
			continue
		}

		if matches := tableHeaderRegex.FindStringSubmatch(line); matches != nil {
			keyName := matches[1]
			rawCols := strings.Split(matches[3], ",")
			columns := make([]string, len(rawCols))
			for idx, c := range rawCols {
				columns[idx] = strings.TrimSpace(c)
			}

			var rows []map[string]interface{}
			i++
			for i < totalLines {
				curr := lines[i]
				currStripped := strings.TrimSpace(curr)
				if currStripped == "" || strings.HasPrefix(currStripped, "#") {
					i++
					continue
				}
				if !strings.HasPrefix(curr, " ") && !strings.HasPrefix(curr, "\t") {
					break
				}

				currClean := stripInlineComment(currStripped)
				reader := csv.NewReader(strings.NewReader(currClean))
				reader.TrimLeadingSpace = true
				record, err := reader.Read()
				if err != nil {
					record = strings.Split(currClean, ",")
				}

				rowDict := make(map[string]interface{}, len(columns))
				for idx, col := range columns {
					if idx < len(record) {
						rowDict[col] = parseScalar(strings.TrimSpace(record[idx]))
					} else {
						rowDict[col] = nil
					}
				}
				rows = append(rows, rowDict)
				i++
			}
			result[keyName] = rows
			continue
		}

		if colonIdx := strings.Index(line, ":"); colonIdx != -1 {
			key := strings.TrimSpace(line[:colonIdx])
			valPart := strings.TrimSpace(stripInlineComment(line[colonIdx+1:]))

			if strings.HasPrefix(valPart, "[") && strings.HasSuffix(valPart, "]") {
				inner := strings.TrimSpace(valPart[1 : len(valPart)-1])
				var items []interface{}
				if inner != "" {
					for _, part := range strings.Split(inner, ",") {
						trimmed := strings.TrimSpace(part)
						if trimmed != "" {
							items = append(items, parseScalar(trimmed))
						}
					}
				} else {
					items = []interface{}{}
				}
				result[key] = items
				i++
				continue
			}

			if valPart != "" {
				result[key] = parseScalar(valPart)
				i++
				continue
			}

			var nestedBlock []string
			i++
			for i < totalLines {
				curr := lines[i]
				currStripped := strings.TrimSpace(curr)
				if currStripped == "" {
					nestedBlock = append(nestedBlock, curr)
					i++
					continue
				}
				if strings.HasPrefix(currStripped, "#") {
					if strings.HasPrefix(curr, " ") || strings.HasPrefix(curr, "\t") {
						nestedBlock = append(nestedBlock, curr)
						i++
						continue
					}
					break
				}
				if !strings.HasPrefix(curr, " ") && !strings.HasPrefix(curr, "\t") {
					break
				}
				nestedBlock = append(nestedBlock, curr)
				i++
			}

			if len(nestedBlock) > 0 {
				dedented := dedentLines(nestedBlock)
				result[key] = ParseMap(dedented)
			} else {
				result[key] = nil
			}
			continue
		}

		i++
	}

	return result
}

func stripInlineComment(val string) string {
	inQuotes := false
	var quoteChar rune
	runes := []rune(val)
	for idx, ch := range runes {
		if ch == '"' || ch == '\'' {
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
			} else if quoteChar == ch {
				inQuotes = false
			}
		} else if ch == '#' && !inQuotes {
			if idx == 0 || runes[idx-1] == ' ' || runes[idx-1] == '\t' {
				return strings.TrimRight(string(runes[:idx]), " \t")
			}
		}
	}
	return val
}

func parseScalar(val string) interface{} {
	val = strings.TrimSpace(stripInlineComment(val))
	if val == "" {
		return nil
	}
	lower := strings.ToLower(val)
	if lower == "true" {
		return true
	}
	if lower == "false" {
		return false
	}
	if lower == "null" || lower == "none" {
		return nil
	}
	if intRegex.MatchString(val) {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	if floatRegex.MatchString(val) {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	if (strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) ||
		(strings.HasPrefix(val, `'`) && strings.HasSuffix(val, `'`)) {
		return val[1 : len(val)-1]
	}
	return val
}

func dedentLines(lines []string) string {
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := 0
		for _, ch := range line {
			if ch == ' ' {
				indent++
			} else if ch == '\t' {
				indent += 2
			} else {
				break
			}
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return strings.Join(lines, "\n")
	}

	out := make([]string, len(lines))
	for i, line := range lines {
		if len(line) >= minIndent {
			out[i] = line[minIndent:]
		} else {
			out[i] = strings.TrimSpace(line)
		}
	}
	return strings.Join(out, "\n")
}
