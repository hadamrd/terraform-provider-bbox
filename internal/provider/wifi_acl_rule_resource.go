package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*wifiACLRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*wifiACLRuleResource)(nil)
	_ resource.ResourceWithImportState = (*wifiACLRuleResource)(nil)
)

// NewWifiACLRuleResource is the resource constructor exposed to the provider.
func NewWifiACLRuleResource() resource.Resource { return &wifiACLRuleResource{} }

type wifiACLRuleResource struct {
	shared *SharedClient
}

type wifiACLRuleModel struct {
	ID         types.Int64  `tfsdk:"id"`
	MacAddress types.String `tfsdk:"macaddress"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

func (r *wifiACLRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wifi_acl_rule"
}

func (r *wifiACLRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A WiFi MAC access-control (ACL) entry. The global filter is toggled " +
			"separately via bbox_wifi_acl; entries only take effect while that is enabled.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.Int64Attribute{Computed: true, Description: "Router-assigned entry ID."},
			"macaddress": schema.StringAttribute{Required: true, Description: "MAC address (aa:bb:cc:dd:ee:ff)."},
			"enabled": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Whether this entry is active.",
			},
		},
	}
}

func (r *wifiACLRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

func (r *wifiACLRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_wifi_acl_rule.")
		return
	}
	var plan wifiACLRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := r.shared.Client.WifiACLAddRule(normaliseMAC(plan.MacAddress.ValueString()), plan.Enabled.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("WiFi ACL add failed", err.Error())
		return
	}
	plan.ID = types.Int64Value(int64(id))
	plan.MacAddress = types.StringValue(normaliseMAC(plan.MacAddress.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wifiACLRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state wifiACLRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rules, err := r.shared.Client.WifiACLRules()
	if err != nil {
		resp.Diagnostics.AddError("WiFi ACL list failed", err.Error())
		return
	}
	wanted := int(state.ID.ValueInt64())
	for _, rAny := range rules {
		m, _ := rAny.(map[string]any)
		if toInt(m["id"]) != wanted {
			continue
		}
		state.MacAddress = types.StringValue(normaliseMAC(toStr(m["macaddress"])))
		state.Enabled = types.BoolValue(toBool(m["enable"]))
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	// drift: entry vanished on the router.
	resp.State.RemoveResource(ctx)
}

func (r *wifiACLRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_wifi_acl_rule.")
		return
	}
	var state, plan wifiACLRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The ACL entry endpoint has no in-place update, so delete + recreate.
	if id := int(state.ID.ValueInt64()); id != 0 {
		if err := r.shared.Client.WifiACLDelRule(id); err != nil {
			resp.Diagnostics.AddError("WiFi ACL delete-for-update failed", err.Error())
			return
		}
	}
	id, err := r.shared.Client.WifiACLAddRule(normaliseMAC(plan.MacAddress.ValueString()), plan.Enabled.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("WiFi ACL recreate failed", err.Error())
		return
	}
	plan.ID = types.Int64Value(int64(id))
	plan.MacAddress = types.StringValue(normaliseMAC(plan.MacAddress.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wifiACLRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.shared == nil {
		return
	}
	var state wifiACLRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if id := int(state.ID.ValueInt64()); id != 0 {
		if err := r.shared.Client.WifiACLDelRule(id); err != nil {
			resp.Diagnostics.AddError("WiFi ACL delete failed", err.Error())
		}
	}
}

func (r *wifiACLRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importInt64ID(ctx, req, resp)
}
