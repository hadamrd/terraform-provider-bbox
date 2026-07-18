package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestWifiBandResourceMetadata(t *testing.T) {
	r := NewWifiBandResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "bbox"}, resp)
	if resp.TypeName != "bbox_wifi_band" {
		t.Fatalf("want bbox_wifi_band, got %q", resp.TypeName)
	}
}

func TestWifiBandResourceSchema(t *testing.T) {
	r := NewWifiBandResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	band, ok := resp.Schema.Attributes["band"]
	if !ok || !band.IsRequired() {
		t.Fatalf("band must exist and be required")
	}
	for _, k := range []string{"enabled", "ssid", "passphrase", "channel"} {
		if _, ok := resp.Schema.Attributes[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
	pw, ok := resp.Schema.Attributes["passphrase"]
	if !ok {
		t.Fatalf("passphrase missing")
	}
	if !pw.IsSensitive() {
		t.Errorf("passphrase must be sensitive")
	}
	if !resp.Schema.Attributes["id"].IsComputed() {
		t.Errorf("id must be computed")
	}
}

func TestValidateBand(t *testing.T) {
	for _, b := range []string{"24", "5", "6"} {
		if !validateBand(b) {
			t.Errorf("expected %q valid", b)
		}
	}
	for _, b := range []string{"", "2.4", "5g", "60"} {
		if validateBand(b) {
			t.Errorf("expected %q invalid", b)
		}
	}
}
