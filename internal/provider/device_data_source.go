package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*deviceDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*deviceDataSource)(nil)
)

// NewDeviceDataSource is the data source constructor exposed to the provider.
func NewDeviceDataSource() datasource.DataSource { return &deviceDataSource{} }

type deviceDataSource struct {
	shared *SharedClient
}

type deviceModel struct {
	Model         types.String `tfsdk:"model"`
	Serial        types.String `tfsdk:"serial"`
	Firmware      types.String `tfsdk:"firmware"`
	UptimeSeconds types.Int64  `tfsdk:"uptime_seconds"`
}

func (d *deviceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (d *deviceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Router metadata (model, serial, firmware, uptime).",
		Attributes: map[string]schema.Attribute{
			"model":          schema.StringAttribute{Computed: true},
			"serial":         schema.StringAttribute{Computed: true},
			"firmware":       schema.StringAttribute{Computed: true},
			"uptime_seconds": schema.Int64Attribute{Computed: true},
		},
	}
}

func (d *deviceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	d.shared = s
}

func (d *deviceDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot read bbox_device.")
		return
	}
	dev, err := d.shared.Client.Device()
	if err != nil {
		resp.Diagnostics.AddError("Device read failed", err.Error())
		return
	}
	// firmware sometimes lives at main.version; fall back to top-level.
	fw := ""
	if m, ok := dev["main"].(map[string]any); ok {
		fw = toStr(m["version"])
	}
	if fw == "" {
		fw = toStr(dev["firmware"])
	}
	out := deviceModel{
		Model:         types.StringValue(toStr(dev["modelname"])),
		Serial:        types.StringValue(toStr(dev["serialnumber"])),
		Firmware:      types.StringValue(fw),
		UptimeSeconds: types.Int64Value(int64(toInt(dev["uptime"]))),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
