package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*dynDNSResource)(nil)
	_ resource.ResourceWithConfigure   = (*dynDNSResource)(nil)
	_ resource.ResourceWithImportState = (*dynDNSResource)(nil)
)

// NewDynDNSResource is the resource constructor exposed to the provider.
func NewDynDNSResource() resource.Resource { return &dynDNSResource{} }

type dynDNSResource struct {
	shared *SharedClient
}

type dynDNSModel struct {
	ID           types.String `tfsdk:"id"`
	ProviderName types.String `tfsdk:"provider_name"`
	Hostname     types.String `tfsdk:"hostname"`
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	Enabled      types.Bool   `tfsdk:"enabled"`
}

func (r *dynDNSResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dyndns"
}

func (r *dynDNSResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Singleton config for the router's DynDNS service.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "Always \"singleton\"."},
			"provider_name": schema.StringAttribute{
				Required:    true,
				Description: "duckdns/dyndns/no-ip/ovh/duiadns/changeip.",
				Validators: []validator.String{
					stringvalidator.OneOf("duckdns", "dyndns", "no-ip", "ovh", "duiadns", "changeip"),
				},
			},
			"hostname": schema.StringAttribute{Required: true},
			"username": schema.StringAttribute{
				Optional: true, Computed: true,
				Default:     stringdefault.StaticString(""),
				Description: "Empty for DuckDNS.",
			},
			"password": schema.StringAttribute{Required: true, Sensitive: true, Description: "DuckDNS: the token from duckdns.org."},
			"enabled": schema.BoolAttribute{
				Optional: true, Computed: true,
				Default: booldefault.StaticBool(true),
			},
		},
	}
}

func (r *dynDNSResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

func (r *dynDNSResource) apply(m dynDNSModel) error {
	if !m.Enabled.ValueBool() {
		return r.shared.Client.DynDNSDisable()
	}
	return r.shared.Client.DynDNSEnable(
		m.ProviderName.ValueString(),
		m.Hostname.ValueString(),
		m.Username.ValueString(),
		m.Password.ValueString(),
	)
}

func (r *dynDNSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_dyndns.")
		return
	}
	var plan dynDNSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(plan); err != nil {
		resp.Diagnostics.AddError("DynDNS apply failed", err.Error())
		return
	}
	plan.ID = types.StringValue("singleton")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dynDNSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state dynDNSModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := r.shared.Client.DynDNS()
	if err != nil {
		resp.Diagnostics.AddError("DynDNS read failed", err.Error())
		return
	}
	// Router redacts the password; keep the state value.
	state.ID = types.StringValue("singleton")
	if v := toStr(raw["server"]); v != "" {
		state.ProviderName = types.StringValue(v)
	}
	if v := toStr(raw["hostname"]); v != "" {
		state.Hostname = types.StringValue(v)
	}
	state.Username = types.StringValue(toStr(raw["username"]))
	state.Enabled = types.BoolValue(toBool(raw["enable"]))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dynDNSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_dyndns.")
		return
	}
	var plan dynDNSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(plan); err != nil {
		resp.Diagnostics.AddError("DynDNS apply failed", err.Error())
		return
	}
	plan.ID = types.StringValue("singleton")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dynDNSResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.shared == nil {
		return
	}
	if err := r.shared.Client.DynDNSDisable(); err != nil {
		resp.Diagnostics.AddError("DynDNS disable failed", err.Error())
	}
}

func (r *dynDNSResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
