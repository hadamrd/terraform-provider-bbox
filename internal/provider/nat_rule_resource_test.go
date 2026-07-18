package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNATRuleResourceMetadata(t *testing.T) {
	r := NewNATRuleResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "bbox"}, resp)
	if resp.TypeName != "bbox_nat_rule" {
		t.Fatalf("want bbox_nat_rule, got %q", resp.TypeName)
	}
}

func TestNATRuleResourceSchema(t *testing.T) {
	r := NewNATRuleResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	required := []string{"name", "external_port", "target_ip"}
	for _, k := range required {
		attr, ok := resp.Schema.Attributes[k]
		if !ok {
			t.Fatalf("missing attribute %q", k)
		}
		if !attr.IsRequired() {
			t.Errorf("%q should be required", k)
		}
	}
	if _, ok := resp.Schema.Attributes["id"]; !ok {
		t.Fatalf("missing computed id")
	}
	if !resp.Schema.Attributes["id"].IsComputed() {
		t.Errorf("id must be computed")
	}
	for _, k := range []string{"protocol", "remote_ip", "skip_port_check", "internal_port"} {
		if _, ok := resp.Schema.Attributes[k]; !ok {
			t.Errorf("missing optional attribute %q", k)
		}
	}
}
