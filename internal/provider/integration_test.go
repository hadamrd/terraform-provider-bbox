package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hadamrd/bbox-cli/pkg/client"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ── Mock router ────────────────────────────────────────────────────────────

// recordedReq captures a single HTTP call the client made against the mock.
type recordedReq struct {
	Method string
	Path   string
	Form   url.Values // parsed application/x-www-form-urlencoded body
}

// mockRouter is a stateful httptest server mimicking the Bbox admin API. It
// tracks NAT and DHCP entries and records every write so tests can assert on
// method/path/body without depending on the (opaque) client-side wire format.
type mockRouter struct {
	server *httptest.Server

	mu           sync.Mutex
	natRules     []map[string]any
	nextNATID    int64
	dhcpClients  []map[string]any
	nextDHCPID   int64
	firewall     []map[string]any
	nextFWID     int64
	writes       []recordedReq
	requestCount atomic.Int64
}

func newMockRouter(t *testing.T) *mockRouter {
	t.Helper()
	m := &mockRouter{nextNATID: 100, nextDHCPID: 200, nextFWID: 300}
	mux := http.NewServeMux()

	// Auth surface — provider Configure calls EnsureAuth which hits /device.
	mux.HandleFunc("/api/v1/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "BBOX_ID", Value: "fake", Path: "/", HttpOnly: true})
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/v1/device", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []any{map[string]any{"device": map[string]any{
			"modelname": "MockBbox", "main": map[string]any{"version": "0"},
		}}})
	})
	mux.HandleFunc("/api/v1/device/token", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []any{map[string]any{"device": map[string]any{
			"token": "tok", "expires": time.Now().Add(5 * time.Minute).UTC().Format(time.RFC3339),
		}}})
	})
	mux.HandleFunc("/api/v1/wan/ip", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []any{map[string]any{"wan": map[string]any{"ip": map[string]any{
			"address": "1.2.3.4", "portrange": "40000:50000",
		}}}})
	})

	mux.HandleFunc("/api/v1/nat/rules", m.natRulesHandler)
	mux.HandleFunc("/api/v1/nat/rules/", m.natRuleByIDHandler)
	mux.HandleFunc("/api/v1/dhcp/clients", m.dhcpClientsHandler)
	mux.HandleFunc("/api/v1/dhcp/clients/", m.dhcpClientByIDHandler)
	mux.HandleFunc("/api/v1/firewall/rules", m.firewallRulesHandler)
	mux.HandleFunc("/api/v1/firewall/rules/", m.firewallRuleByIDHandler)

	m.server = httptest.NewTLSServer(mux)
	return m
}

func (m *mockRouter) record(r *http.Request) url.Values {
	m.requestCount.Add(1)
	if r.Method == http.MethodGet {
		return nil
	}
	body, _ := io.ReadAll(r.Body)
	form, _ := url.ParseQuery(string(body))
	m.mu.Lock()
	m.writes = append(m.writes, recordedReq{Method: r.Method, Path: r.URL.Path, Form: form})
	m.mu.Unlock()
	return form
}

func (m *mockRouter) natRulesHandler(w http.ResponseWriter, r *http.Request) {
	form := m.record(r)
	m.mu.Lock()
	defer m.mu.Unlock()
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, []any{map[string]any{"nat": map[string]any{
			"enable": float64(1),
			"rules":  toAnyList(m.natRules),
		}}})
	case http.MethodPost:
		m.nextNATID++
		id := m.nextNATID
		rule := map[string]any{
			"id":           float64(id),
			"description":  form.Get("description"),
			"protocol":     form.Get("protocol"),
			"externalport": floatOr(form.Get("external_port")),
			"internalport": floatOr(form.Get("internal_port")),
			"internalip":   form.Get("ipaddress"),
			"ipremote":     form.Get("ipremote"),
			"enable":       floatOr(form.Get("enable")),
		}
		m.natRules = append(m.natRules, rule)
		w.Header().Set("Location", fmt.Sprintf("/api/v1/nat/rules/%d", id))
		w.WriteHeader(http.StatusCreated)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (m *mockRouter) natRuleByIDHandler(w http.ResponseWriter, r *http.Request) {
	form := m.record(r)
	id := lastPathSegmentInt(r.URL.Path)
	m.mu.Lock()
	defer m.mu.Unlock()
	switch r.Method {
	case http.MethodDelete:
		m.natRules = removeByID(m.natRules, id)
		w.WriteHeader(http.StatusOK)
	case http.MethodPut:
		for _, ru := range m.natRules {
			if int64(ru["id"].(float64)) == id {
				if v := form.Get("enable"); v != "" {
					ru["enable"] = floatOr(v)
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (m *mockRouter) dhcpClientsHandler(w http.ResponseWriter, r *http.Request) {
	form := m.record(r)
	m.mu.Lock()
	defer m.mu.Unlock()
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, []any{map[string]any{"clients": toAnyList(m.dhcpClients)}})
	case http.MethodPost:
		m.nextDHCPID++
		id := m.nextDHCPID
		entry := map[string]any{
			"id":         float64(id),
			"macaddress": form.Get("macaddress"),
			"ipaddress":  form.Get("ipaddress"),
			"hostname":   form.Get("hostname"),
		}
		m.dhcpClients = append(m.dhcpClients, entry)
		w.Header().Set("Location", fmt.Sprintf("/api/v1/dhcp/clients/%d", id))
		w.WriteHeader(http.StatusCreated)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (m *mockRouter) dhcpClientByIDHandler(w http.ResponseWriter, r *http.Request) {
	m.record(r)
	id := lastPathSegmentInt(r.URL.Path)
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.Method == http.MethodDelete {
		m.dhcpClients = removeByID(m.dhcpClients, id)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "method", http.StatusMethodNotAllowed)
}

func (m *mockRouter) firewallRulesHandler(w http.ResponseWriter, r *http.Request) {
	form := m.record(r)
	m.mu.Lock()
	defer m.mu.Unlock()
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, []any{map[string]any{"firewall": map[string]any{"rules": toAnyList(m.firewall)}}})
	case http.MethodPost:
		m.nextFWID++
		id := m.nextFWID
		rule := map[string]any{
			"id":          float64(id),
			"description": form.Get("description"),
			"action":      form.Get("action"),
			"protocol":    form.Get("protocol"),
			"dstip":       form.Get("dstip"),
			"dstport":     form.Get("dstport"),
			"srcip":       form.Get("srcip"),
			"srcport":     form.Get("srcport"),
			"ipprotocol":  form.Get("ipprotocol"),
			"enable":      floatOr(form.Get("enable")),
		}
		m.firewall = append(m.firewall, rule)
		w.Header().Set("Location", fmt.Sprintf("/api/v1/firewall/rules/%d", id))
		w.WriteHeader(http.StatusCreated)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (m *mockRouter) firewallRuleByIDHandler(w http.ResponseWriter, r *http.Request) {
	form := m.record(r)
	id := lastPathSegmentInt(r.URL.Path)
	m.mu.Lock()
	defer m.mu.Unlock()
	switch r.Method {
	case http.MethodDelete:
		m.firewall = removeByID(m.firewall, id)
		w.WriteHeader(http.StatusOK)
	case http.MethodPut:
		for _, ru := range m.firewall {
			if int64(ru["id"].(float64)) == id {
				if v := form.Get("enable"); v != "" {
					ru["enable"] = floatOr(v)
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (m *mockRouter) countByMethodPath(method, path string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, w := range m.writes {
		if w.Method == method && w.Path == path {
			n++
		}
	}
	return n
}

// ── helpers ────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(v)
	_, _ = w.Write(b)
}

func toAnyList(ms []map[string]any) []any {
	out := make([]any, len(ms))
	for i, m := range ms {
		out[i] = m
	}
	return out
}

func floatOr(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(s, "%f", &f)
	return f
}

func lastPathSegmentInt(p string) int64 {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return 0
	}
	var n int64
	_, _ = fmt.Sscanf(p[i+1:], "%d", &n)
	return n
}

func removeByID(ms []map[string]any, id int64) []map[string]any {
	out := ms[:0]
	for _, m := range ms {
		if int64(m["id"].(float64)) != id {
			out = append(out, m)
		}
	}
	return out
}

// ── Test harness ───────────────────────────────────────────────────────────

// setupIntegration boots a mock router, isolates HOME so the session file
// lands in a temp dir, points client.BaseURL at the mock, and returns the
// mock + a per-test provider config. Uses resource.UnitTest so no TF_ACC=1.
func setupIntegration(t *testing.T) (*mockRouter, string) {
	t.Helper()
	m := newMockRouter(t)
	t.Cleanup(func() { m.server.Close() })

	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	} else {
		t.Setenv("HOME", dir)
	}
	if err := os.WriteFile(filepath.Join(dir, ".bbox-password"), []byte("mock"), 0600); err != nil {
		t.Fatalf("write pw: %v", err)
	}
	// Provider Configure calls Login which writes to SessionFile(); point that
	// at a per-test file so parallel tests don't clash.
	client.SetSessionFile(filepath.Join(dir, ".bbox-session.json"))
	t.Cleanup(func() { client.SetSessionFile("") })

	prev := client.BaseURL
	client.BaseURL = m.server.URL
	t.Cleanup(func() { client.BaseURL = prev })

	providerBlock := fmt.Sprintf(`provider "bbox" {
  base_url      = %q
  password_file = %q
}
`, m.server.URL, filepath.Join(dir, ".bbox-password"))
	return m, providerBlock
}

func providerFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"bbox": providerserver.NewProtocol6WithError(New("test")()),
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestIntegration_NATRuleCreate(t *testing.T) {
	m, prov := setupIntegration(t)
	cfg := prov + `
resource "bbox_nat_rule" "r" {
  name            = "test-forward"
  external_port   = 45000
  target_ip       = "192.168.1.10"
  protocol        = "tcp"
  skip_port_check = true
}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{{
			Config: cfg,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet("bbox_nat_rule.r", "id"),
				resource.TestCheckResourceAttr("bbox_nat_rule.r", "protocol", "tcp"),
				resource.TestCheckResourceAttr("bbox_nat_rule.r", "enabled", "true"),
			),
		}},
	})
	if got := m.countByMethodPath("POST", "/api/v1/nat/rules"); got != 1 {
		t.Errorf("expected 1 POST /nat/rules, got %d", got)
	}
}

func TestIntegration_NATRuleUpdatePortRecreates(t *testing.T) {
	m, prov := setupIntegration(t)
	tpl := prov + `
resource "bbox_nat_rule" "r" {
  name            = "recreate"
  external_port   = %d
  target_ip       = "192.168.1.10"
  skip_port_check = true
}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{Config: fmt.Sprintf(tpl, 45010)},
			{Config: fmt.Sprintf(tpl, 45011),
				Check: resource.TestCheckResourceAttr("bbox_nat_rule.r", "external_port", "45011")},
		},
	})
	if got := m.countByMethodPath("POST", "/api/v1/nat/rules"); got != 2 {
		t.Errorf("expected 2 POSTs (create + update-recreate), got %d", got)
	}
	if got, want := m.countByMethodPath("DELETE", "/api/v1/nat/rules/101"), 1; got != want {
		t.Errorf("expected %d DELETE of first rule, got %d", want, got)
	}
}

func TestIntegration_NATRuleUpdateEnabledOnlyToggles(t *testing.T) {
	m, prov := setupIntegration(t)
	tpl := prov + `
resource "bbox_nat_rule" "r" {
  name            = "toggle"
  external_port   = 45020
  target_ip       = "192.168.1.10"
  skip_port_check = true
  enabled         = %t
}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{Config: fmt.Sprintf(tpl, true)},
			{Config: fmt.Sprintf(tpl, false),
				Check: resource.TestCheckResourceAttr("bbox_nat_rule.r", "enabled", "false")},
		},
	})
	// Enabled-only should skip the destroy+recreate: exactly 1 POST (create), 1
	// PUT (in-place toggle), and 1 DELETE (implicit test cleanup at the end).
	if got := m.countByMethodPath("POST", "/api/v1/nat/rules"); got != 1 {
		t.Errorf("expected 1 POST (enabled-only toggle should not recreate), got %d", got)
	}
	if got := m.countByMethodPath("PUT", "/api/v1/nat/rules/101"); got != 1 {
		t.Errorf("expected 1 PUT (in-place toggle), got %d", got)
	}
	if got := m.countByMethodPath("DELETE", "/api/v1/nat/rules/101"); got != 1 {
		t.Errorf("expected 1 DELETE (test cleanup only), got %d", got)
	}
}

func TestIntegration_NATRuleDelete(t *testing.T) {
	m, prov := setupIntegration(t)
	cfg := prov + `
resource "bbox_nat_rule" "r" {
  name            = "delme"
  external_port   = 45030
  target_ip       = "192.168.1.10"
  skip_port_check = true
}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps:                    []resource.TestStep{{Config: cfg, Destroy: false}},
	})
	// resource.UnitTest always destroys at the end.
	if got := m.countByMethodPath("DELETE", "/api/v1/nat/rules/101"); got != 1 {
		t.Errorf("expected 1 DELETE after implicit destroy, got %d", got)
	}
}

func TestIntegration_DHCPReservationLifecycle(t *testing.T) {
	m, prov := setupIntegration(t)
	tpl := prov + `
resource "bbox_dhcp_reservation" "r" {
  mac        = "aa:bb:cc:dd:ee:ff"
  ip_address = "%s"
}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{Config: fmt.Sprintf(tpl, "192.168.1.50")},
			{Config: fmt.Sprintf(tpl, "192.168.1.51"),
				Check: resource.TestCheckResourceAttr("bbox_dhcp_reservation.r", "ip_address", "192.168.1.51")},
		},
	})
	if got := m.countByMethodPath("POST", "/api/v1/dhcp/clients"); got != 2 {
		t.Errorf("expected 2 POSTs (create + update-recreate), got %d", got)
	}
	if got := m.countByMethodPath("DELETE", "/api/v1/dhcp/clients/201"); got != 1 {
		t.Errorf("expected 1 DELETE of first reservation, got %d", got)
	}
}

func TestIntegration_FirewallRuleEnabledOnlyToggles(t *testing.T) {
	m, prov := setupIntegration(t)
	tpl := prov + `
resource "bbox_firewall_rule" "r" {
  name     = "block-ssh"
  action   = "Drop"
  protocol = "tcp"
  dst_port = "22"
  enabled  = %t
}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{Config: fmt.Sprintf(tpl, true)},
			{Config: fmt.Sprintf(tpl, false),
				Check: resource.TestCheckResourceAttr("bbox_firewall_rule.r", "enabled", "false")},
		},
	})
	if got := m.countByMethodPath("POST", "/api/v1/firewall/rules"); got != 1 {
		t.Errorf("expected 1 POST (toggle should not recreate), got %d", got)
	}
	if got := m.countByMethodPath("PUT", "/api/v1/firewall/rules/301"); got != 1 {
		t.Errorf("expected 1 PUT (in-place toggle), got %d", got)
	}
}

func TestIntegration_InvalidProtocolFailsPlan(t *testing.T) {
	_, prov := setupIntegration(t)
	cfg := prov + `
resource "bbox_nat_rule" "r" {
  name            = "bad"
  external_port   = 45040
  target_ip       = "192.168.1.10"
  protocol        = "sctp"   # invalid
  skip_port_check = true
}`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{{
			Config:      cfg,
			ExpectError: regexp.MustCompile(`(?is)value must be one of`),
		}},
	})
}

// suppress unused-import warning for context if package-init tightens later.
var _ = context.Background
var _ = strings.Contains
