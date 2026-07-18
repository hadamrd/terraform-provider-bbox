package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestWANDataSourceSchema(t *testing.T) {
	d := NewWANDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	for _, k := range []string{"ip_v4", "ip_v6", "state", "mac", "port_range", "port_range_low", "port_range_high", "map_t_enabled"} {
		if _, ok := resp.Schema.Attributes[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
}

func TestHostDataSourceSchema(t *testing.T) {
	d := NewHostDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	for _, k := range []string{"id", "hostname", "ip_address", "mac", "link", "active"} {
		if _, ok := resp.Schema.Attributes[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
}

func TestHostsDataSourceSchema(t *testing.T) {
	d := NewHostsDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	for _, k := range []string{"active_only", "hosts"} {
		if _, ok := resp.Schema.Attributes[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
}

func TestDeviceDataSourceSchema(t *testing.T) {
	d := NewDeviceDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diags: %v", resp.Diagnostics)
	}
	for _, k := range []string{"model", "serial", "firmware", "uptime_seconds"} {
		if _, ok := resp.Schema.Attributes[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
}

func TestMetadata_All(t *testing.T) {
	cases := map[string]datasource.DataSource{
		"bbox_wan":    NewWANDataSource(),
		"bbox_host":   NewHostDataSource(),
		"bbox_hosts":  NewHostsDataSource(),
		"bbox_device": NewDeviceDataSource(),
	}
	for want, ds := range cases {
		resp := &datasource.MetadataResponse{}
		ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "bbox"}, resp)
		if resp.TypeName != want {
			t.Errorf("got %q, want %q", resp.TypeName, want)
		}
	}
}
