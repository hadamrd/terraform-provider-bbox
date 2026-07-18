package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestHostResourceMetadata(t *testing.T) {
	r := NewHostResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "bbox"}, resp)
	if resp.TypeName != "bbox_host" {
		t.Fatalf("want bbox_host, got %q", resp.TypeName)
	}
}

func TestHostResourceSchema(t *testing.T) {
	r := NewHostResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	mac, ok := resp.Schema.Attributes["mac"]
	if !ok || !mac.IsRequired() {
		t.Fatalf("mac must be required")
	}
	for _, k := range []string{"hostname", "blocked"} {
		if _, ok := resp.Schema.Attributes[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
	if !resp.Schema.Attributes["id"].IsComputed() {
		t.Errorf("id must be computed")
	}
}
