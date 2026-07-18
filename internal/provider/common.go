package provider

import (
	"fmt"
	"strconv"
	"strings"
)

// sharedFromReq extracts the *SharedClient from a resource/datasource Configure
// providerData. Returns nil on the initial nil-data call so resources don't spam
// errors before Configure runs.
func sharedFromAny(v any) (*SharedClient, error) {
	if v == nil {
		return nil, nil
	}
	s, ok := v.(*SharedClient)
	if !ok {
		return nil, fmt.Errorf("unexpected provider data type %T", v)
	}
	return s, nil
}

// toInt normalises a JSON-decoded int (float64/int/string) to int.
func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}

// toStr normalises a JSON-decoded string.
func toStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case bool:
		return strconv.FormatBool(x)
	}
	return ""
}

// toBool normalises firmware bool encodings (bool, "1"/"0", 1/0).
func toBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case int:
		return x != 0
	case string:
		return x == "1" || strings.EqualFold(x, "true")
	}
	return false
}

// normaliseMAC lowercases + converts dashes to colons.
func normaliseMAC(mac string) string {
	return strings.ToLower(strings.ReplaceAll(mac, "-", ":"))
}

// parsePortRange parses "40960:49151" into (low, high). Returns (0,0) on empty
// or unparseable input (which means "full range" per firmware convention).
func parsePortRange(s string) (int, int) {
	if !strings.Contains(s, ":") {
		return 0, 0
	}
	parts := strings.SplitN(s, ":", 2)
	lo, err1 := strconv.Atoi(parts[0])
	hi, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	return lo, hi
}

// walkDHCPClients flattens the loose /dhcp/clients firmware shape into a slice
// of maps. Mirrors filterDHCP in bbox-cli.
func walkDHCPClients(root map[string]any) []map[string]any {
	var out []map[string]any
	visit := func(arr []any) {
		for _, e := range arr {
			if m, ok := e.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	for _, v := range root {
		if arr, ok := v.([]any); ok {
			visit(arr)
			continue
		}
		if inner, ok := v.(map[string]any); ok {
			for _, iv := range inner {
				if arr, ok := iv.([]any); ok {
					visit(arr)
				}
			}
		}
	}
	return out
}
