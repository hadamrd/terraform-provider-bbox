package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDynDNSResourceMetadata(t *testing.T) {
	r := NewDynDNSResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "bbox"}, resp)
	if resp.TypeName != "bbox_dyndns" {
		t.Fatalf("want bbox_dyndns, got %q", resp.TypeName)
	}
}

func TestDynDNSResourceSchema(t *testing.T) {
	r := NewDynDNSResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	for _, k := range []string{"provider_name", "hostname", "password"} {
		attr, ok := resp.Schema.Attributes[k]
		if !ok {
			t.Fatalf("missing %q", k)
		}
		if !attr.IsRequired() {
			t.Errorf("%q must be required", k)
		}
	}
	pw, ok := resp.Schema.Attributes["password"]
	if !ok || !pw.IsSensitive() {
		t.Errorf("password must be sensitive")
	}
	for _, k := range []string{"username", "enabled"} {
		if _, ok := resp.Schema.Attributes[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
	if !resp.Schema.Attributes["id"].IsComputed() {
		t.Errorf("id must be computed")
	}
}
