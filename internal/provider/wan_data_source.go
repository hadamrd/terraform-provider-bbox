package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*wanDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*wanDataSource)(nil)
)

// NewWANDataSource is the data source constructor exposed to the provider.
func NewWANDataSource() datasource.DataSource { return &wanDataSource{} }

type wanDataSource struct {
	shared *SharedClient
}

type wanModel struct {
	IPv4          types.String `tfsdk:"ip_v4"`
	IPv6          types.String `tfsdk:"ip_v6"`
	State         types.String `tfsdk:"state"`
	MAC           types.String `tfsdk:"mac"`
	PortRange     types.String `tfsdk:"port_range"`
	PortRangeLow  types.Int64  `tfsdk:"port_range_low"`
	PortRangeHigh types.Int64  `tfsdk:"port_range_high"`
	MAPTEnabled   types.Bool   `tfsdk:"map_t_enabled"`
}

func (d *wanDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wan"
}

func (d *wanDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read the current WAN state (public IP, MAP-T range, etc).",
		Attributes: map[string]schema.Attribute{
			"ip_v4":           schema.StringAttribute{Computed: true, Description: "Public IPv4."},
			"ip_v6":           schema.StringAttribute{Computed: true, Description: "First public IPv6 or empty."},
			"state":           schema.StringAttribute{Computed: true, Description: "'Up' or 'Down'."},
			"mac":             schema.StringAttribute{Computed: true, Description: "WAN interface MAC."},
			"port_range":      schema.StringAttribute{Computed: true, Description: "MAP-T port range e.g. '40960:49151'."},
			"port_range_low":  schema.Int64Attribute{Computed: true, Description: "Parsed low bound (0 = full)."},
			"port_range_high": schema.Int64Attribute{Computed: true, Description: "Parsed high bound (0 = full)."},
			"map_t_enabled":   schema.BoolAttribute{Computed: true, Description: "MAP-T on/off."},
		},
	}
}

func (d *wanDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	d.shared = s
}

func (d *wanDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot read bbox_wan.")
		return
	}
	wan, err := d.shared.Client.WAN()
	if err != nil {
		resp.Diagnostics.AddError("WAN read failed", err.Error())
		return
	}
	ip, _ := wan["ip"].(map[string]any)
	if ip == nil {
		ip = map[string]any{}
	}

	ipv6 := ""
	if arr, ok := ip["ip6address"].([]any); ok && len(arr) > 0 {
		if m, ok := arr[0].(map[string]any); ok {
			ipv6 = toStr(m["ipaddress"])
		}
	}
	rng := toStr(ip["portrange"])
	lo, hi := parsePortRange(rng)

	out := wanModel{
		IPv4:          types.StringValue(toStr(ip["address"])),
		IPv6:          types.StringValue(ipv6),
		State:         types.StringValue(toStr(ip["state"])),
		MAC:           types.StringValue(toStr(ip["mac"])),
		PortRange:     types.StringValue(rng),
		PortRangeLow:  types.Int64Value(int64(lo)),
		PortRangeHigh: types.Int64Value(int64(hi)),
		MAPTEnabled:   types.BoolValue(toBool(ip["maptenable"])),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
