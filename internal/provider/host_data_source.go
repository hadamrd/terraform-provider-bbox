package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource                   = (*hostDataSource)(nil)
	_ datasource.DataSourceWithConfigure      = (*hostDataSource)(nil)
	_ datasource.DataSourceWithValidateConfig = (*hostDataSource)(nil)
)

// NewHostDataSource is the data source constructor exposed to the provider.
func NewHostDataSource() datasource.DataSource { return &hostDataSource{} }

type hostDataSource struct {
	shared *SharedClient
}

// hostModel is shared with the hosts (list) data source.
type hostModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Hostname  types.String `tfsdk:"hostname"`
	IPAddress types.String `tfsdk:"ip_address"`
	MAC       types.String `tfsdk:"mac"`
	Link      types.String `tfsdk:"link"`
	Active    types.Bool   `tfsdk:"active"`
}

func (d *hostDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (d *hostDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up one LAN host. Set exactly one of id / hostname / mac / ip_address.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.Int64Attribute{Optional: true, Computed: true, Description: "Router-assigned host id."},
			"hostname":   schema.StringAttribute{Optional: true, Computed: true},
			"ip_address": schema.StringAttribute{Optional: true, Computed: true},
			"mac":        schema.StringAttribute{Optional: true, Computed: true},
			"link":       schema.StringAttribute{Computed: true, Description: "'Wifi 5', 'Wifi 2.4', 'Ethernet', 'Offline'."},
			"active":     schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *hostDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	d.shared = s
}

func (d *hostDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var cfg hostModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	set := 0
	if !cfg.ID.IsNull() && !cfg.ID.IsUnknown() {
		set++
	}
	if !cfg.Hostname.IsNull() && !cfg.Hostname.IsUnknown() {
		set++
	}
	if !cfg.MAC.IsNull() && !cfg.MAC.IsUnknown() {
		set++
	}
	if !cfg.IPAddress.IsNull() && !cfg.IPAddress.IsUnknown() {
		set++
	}
	if set != 1 {
		resp.Diagnostics.AddError(
			"Invalid bbox_host lookup",
			"Exactly one of id, hostname, mac, or ip_address must be set.",
		)
	}
}

func (d *hostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot read bbox_host.")
		return
	}
	var cfg hostModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var key string
	switch {
	case !cfg.ID.IsNull():
		key = strconv.FormatInt(cfg.ID.ValueInt64(), 10)
	case !cfg.Hostname.IsNull():
		key = cfg.Hostname.ValueString()
	case !cfg.MAC.IsNull():
		key = cfg.MAC.ValueString()
	case !cfg.IPAddress.IsNull():
		key = cfg.IPAddress.ValueString()
	}
	h, err := d.shared.Client.HostBy(key)
	if err != nil {
		resp.Diagnostics.AddError("Host lookup failed", err.Error())
		return
	}
	out := hostMapToModel(h)
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}

func hostMapToModel(h map[string]any) hostModel {
	return hostModel{
		ID:        types.Int64Value(int64(toInt(h["id"]))),
		Hostname:  types.StringValue(toStr(h["hostname"])),
		IPAddress: types.StringValue(toStr(h["ipaddress"])),
		MAC:       types.StringValue(toStr(h["macaddress"])),
		Link:      types.StringValue(toStr(h["link"])),
		Active:    types.BoolValue(toBool(h["active"])),
	}
}
