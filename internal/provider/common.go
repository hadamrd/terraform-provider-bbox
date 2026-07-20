package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// importInt64ID adopts a resource whose `id` attribute is an Int64. The native
// `import {}` block (and `terraform import`) hand the ID in as a string, which
// ImportStatePassthroughID can't coerce into a number — so parse it explicitly.
func importInt64ID(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(strings.TrimSpace(req.ID), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("expected an integer id, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

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

// canonicalDays is the stable day ordering used for scheduler "occurency"
// conversion. The Bbox indexes Mon=1..Sat=6, Sun=0; we keep Sun last so diffs
// stay stable regardless of the order the user lists days.
var canonicalDays = []struct {
	name  string
	index string
}{
	{"mon", "1"}, {"tue", "2"}, {"wed", "3"}, {"thu", "4"}, {"fri", "5"}, {"sat", "6"}, {"sun", "0"},
}

// daysToOccurency converts a list of lowercase day names into the Bbox
// "occurency" string (comma-separated indices) in canonical order.
func daysToOccurency(days []string) (string, error) {
	want := map[string]bool{}
	for _, d := range days {
		want[strings.ToLower(strings.TrimSpace(d))] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, cd := range canonicalDays {
		if want[cd.name] {
			out = append(out, cd.index)
			seen[cd.name] = true
		}
	}
	for name := range want {
		if !seen[name] {
			return "", fmt.Errorf("invalid day %q (use mon,tue,wed,thu,fri,sat,sun)", name)
		}
	}
	if len(out) == 0 {
		return "", fmt.Errorf("at least one day is required")
	}
	return strings.Join(out, ","), nil
}

// occurencyToDays converts a Bbox "occurency" string back into day names in
// canonical order, for stable round-tripping in state.
func occurencyToDays(occ string) []string {
	has := map[string]bool{}
	for _, tok := range strings.Split(occ, ",") {
		has[strings.TrimSpace(tok)] = true
	}
	var out []string
	for _, cd := range canonicalDays {
		if has[cd.index] {
			out = append(out, cd.name)
		}
	}
	return out
}

// splitIntervals splits a Bbox "HH:MM,HH:MM" interval into (start, end).
func splitIntervals(intervals string) (string, string) {
	parts := strings.SplitN(intervals, ",", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
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
