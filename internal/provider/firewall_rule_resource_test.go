package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFirewallRuleResourceMetadata(t *testing.T) {
	r := NewFirewallRuleResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "bbox"}, resp)
	if resp.TypeName != "bbox_firewall_rule" {
		t.Fatalf("want bbox_firewall_rule, got %q", resp.TypeName)
	}
}

func TestFirewallRuleResourceSchema(t *testing.T) {
	r := NewFirewallRuleResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	for _, k := range []string{"name", "action", "protocol"} {
		attr, ok := resp.Schema.Attributes[k]
		if !ok {
			t.Fatalf("missing attribute %q", k)
		}
		if !attr.IsRequired() {
			t.Errorf("%q should be required", k)
		}
	}
	for _, k := range []string{"dst_ip", "dst_port", "src_ip", "src_port", "enabled", "ip_version"} {
		if _, ok := resp.Schema.Attributes[k]; !ok {
			t.Errorf("missing optional attribute %q", k)
		}
	}
	if !resp.Schema.Attributes["id"].IsComputed() {
		t.Errorf("id must be computed")
	}
}

func TestFirewallOnlyEnabledChanged(t *testing.T) {
	base := firewallRuleModel{
		Name:      types.StringValue("r"),
		Action:    types.StringValue("Drop"),
		Protocol:  types.StringValue("tcp"),
		DstIP:     types.StringValue("1.2.3.4"),
		DstPort:   types.StringValue("80"),
		SrcIP:     types.StringValue(""),
		SrcPort:   types.StringValue(""),
		IPVersion: types.StringValue("IPv4"),
		Enabled:   types.BoolValue(true),
	}
	plan := base
	plan.Enabled = types.BoolValue(false)
	if !firewallOnlyEnabledChanged(base, plan) {
		t.Errorf("expected true when only enabled changed")
	}
	plan2 := base
	plan2.DstPort = types.StringValue("81")
	if firewallOnlyEnabledChanged(base, plan2) {
		t.Errorf("expected false when dst_port changed")
	}
	same := base
	if firewallOnlyEnabledChanged(base, same) {
		t.Errorf("expected false when nothing changed")
	}
}
