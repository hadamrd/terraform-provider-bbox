package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestUPnPResourceMetadata(t *testing.T) {
	r := NewUPnPResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "bbox"}, resp)
	if resp.TypeName != "bbox_upnp" {
		t.Fatalf("want bbox_upnp, got %q", resp.TypeName)
	}
}

func TestUPnPResourceSchema(t *testing.T) {
	r := NewUPnPResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	en, ok := resp.Schema.Attributes["enabled"]
	if !ok || !en.IsRequired() {
		t.Fatalf("enabled must be required")
	}
	if !resp.Schema.Attributes["id"].IsComputed() {
		t.Errorf("id must be computed")
	}
}
