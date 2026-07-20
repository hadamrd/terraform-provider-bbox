package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*wifiACLResource)(nil)
	_ resource.ResourceWithConfigure   = (*wifiACLResource)(nil)
	_ resource.ResourceWithImportState = (*wifiACLResource)(nil)
)

// NewWifiACLResource is the resource constructor exposed to the provider.
func NewWifiACLResource() resource.Resource { return &wifiACLResource{} }

type wifiACLResource struct {
	shared *SharedClient
}

type wifiACLModel struct {
	ID      types.String `tfsdk:"id"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func (r *wifiACLResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wifi_acl"
}

func (r *wifiACLResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Singleton for the global WiFi MAC access-control filter. When enabled, the " +
			"router enforces the bbox_wifi_acl_rule entries. Ensure the managing device is covered " +
			"before enabling or you may lock yourself out of WiFi (wired access is unaffected).",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true, Description: "Always \"singleton\"."},
			"enabled": schema.BoolAttribute{Required: true, Description: "Whether MAC filtering is enforced."},
		},
	}
}

func (r *wifiACLResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

func (r *wifiACLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_wifi_acl.")
		return
	}
	var plan wifiACLModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.shared.Client.WifiACLToggle(plan.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("WiFi ACL toggle failed", err.Error())
		return
	}
	plan.ID = types.StringValue("singleton")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wifiACLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state wifiACLModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	on, err := r.shared.Client.WifiACLEnabled()
	if err != nil {
		resp.Diagnostics.AddError("WiFi ACL read failed", err.Error())
		return
	}
	state.ID = types.StringValue("singleton")
	state.Enabled = types.BoolValue(on)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *wifiACLResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_wifi_acl.")
		return
	}
	var plan wifiACLModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.shared.Client.WifiACLToggle(plan.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("WiFi ACL toggle failed", err.Error())
		return
	}
	plan.ID = types.StringValue("singleton")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wifiACLResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"bbox_wifi_acl destroy is a no-op",
		"Removing this resource only removes it from Terraform state; the router's MAC-filter flag is unchanged.",
	)
}

func (r *wifiACLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
