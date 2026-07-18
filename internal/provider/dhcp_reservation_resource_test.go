package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDHCPReservationResourceMetadata(t *testing.T) {
	r := NewDHCPReservationResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "bbox"}, resp)
	if resp.TypeName != "bbox_dhcp_reservation" {
		t.Fatalf("want bbox_dhcp_reservation, got %q", resp.TypeName)
	}
}

func TestDHCPReservationResourceSchema(t *testing.T) {
	r := NewDHCPReservationResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	for _, k := range []string{"mac", "ip_address"} {
		attr, ok := resp.Schema.Attributes[k]
		if !ok {
			t.Fatalf("missing attribute %q", k)
		}
		if !attr.IsRequired() {
			t.Errorf("%q should be required", k)
		}
	}
	if !resp.Schema.Attributes["id"].IsComputed() {
		t.Errorf("id must be computed")
	}
	if _, ok := resp.Schema.Attributes["hostname"]; !ok {
		t.Errorf("hostname missing")
	}
}

func TestNormaliseMAC(t *testing.T) {
	cases := map[string]string{
		"AA-BB-CC-DD-EE-FF": "aa:bb:cc:dd:ee:ff",
		"aa:bb:cc:dd:ee:ff": "aa:bb:cc:dd:ee:ff",
		"Aa:Bb-Cc:Dd-Ee:Ff": "aa:bb:cc:dd:ee:ff",
	}
	for in, want := range cases {
		if got := normaliseMAC(in); got != want {
			t.Errorf("normaliseMAC(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParsePortRange(t *testing.T) {
	lo, hi := parsePortRange("40960:49151")
	if lo != 40960 || hi != 49151 {
		t.Errorf("got %d:%d", lo, hi)
	}
	lo, hi = parsePortRange("")
	if lo != 0 || hi != 0 {
		t.Errorf("empty want 0:0, got %d:%d", lo, hi)
	}
	lo, hi = parsePortRange("not-a-range")
	if lo != 0 || hi != 0 {
		t.Errorf("garbage want 0:0, got %d:%d", lo, hi)
	}
}
