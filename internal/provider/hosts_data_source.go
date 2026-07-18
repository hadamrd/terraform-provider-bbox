package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*hostsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*hostsDataSource)(nil)
)

// NewHostsDataSource is the data source constructor exposed to the provider.
func NewHostsDataSource() datasource.DataSource { return &hostsDataSource{} }

type hostsDataSource struct {
	shared *SharedClient
}

type hostsModel struct {
	ActiveOnly types.Bool  `tfsdk:"active_only"`
	Hosts      []hostModel `tfsdk:"hosts"`
}

func (d *hostsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hosts"
}

func (d *hostsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all LAN hosts.",
		Attributes: map[string]schema.Attribute{
			"active_only": schema.BoolAttribute{Optional: true, Description: "Filter to hosts where active=true."},
			"hosts": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.Int64Attribute{Computed: true},
						"hostname":   schema.StringAttribute{Computed: true},
						"ip_address": schema.StringAttribute{Computed: true},
						"mac":        schema.StringAttribute{Computed: true},
						"link":       schema.StringAttribute{Computed: true},
						"active":     schema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *hostsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	d.shared = s
}

func (d *hostsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot read bbox_hosts.")
		return
	}
	var cfg hostsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	hosts, err := d.shared.Client.Hosts()
	if err != nil {
		resp.Diagnostics.AddError("Hosts list failed", err.Error())
		return
	}
	activeOnly := cfg.ActiveOnly.ValueBool()
	out := hostsModel{ActiveOnly: cfg.ActiveOnly}
	for _, hAny := range hosts {
		h, _ := hAny.(map[string]any)
		if h == nil {
			continue
		}
		if activeOnly && !toBool(h["active"]) {
			continue
		}
		out.Hosts = append(out.Hosts, hostMapToModel(h))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
