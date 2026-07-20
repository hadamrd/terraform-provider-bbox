package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*dmzResource)(nil)
	_ resource.ResourceWithConfigure   = (*dmzResource)(nil)
	_ resource.ResourceWithImportState = (*dmzResource)(nil)
)

// NewDMZResource is the resource constructor exposed to the provider.
func NewDMZResource() resource.Resource { return &dmzResource{} }

type dmzResource struct {
	shared *SharedClient
}

type dmzModel struct {
	ID       types.String `tfsdk:"id"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	TargetIP types.String `tfsdk:"target_ip"`
}

func (r *dmzResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dmz"
}

func (r *dmzResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Singleton single-host DMZ. Exposes one LAN host to all inbound WAN traffic " +
			"— a security risk. Commonly managed as `enabled = false` to assert (and detect drift on) " +
			"the DMZ staying off.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true, Description: "Always \"singleton\"."},
			"enabled": schema.BoolAttribute{Required: true, Description: "Whether the DMZ is active."},
			"target_ip": schema.StringAttribute{
				Optional: true, Computed: true, Default: stringdefault.StaticString(""),
				Description: "LAN IP exposed when enabled. Required if enabled = true.",
			},
		},
	}
}

func (r *dmzResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

func (r *dmzResource) apply(plan dmzModel) error {
	if plan.Enabled.ValueBool() {
		return r.shared.Client.DMZSet(plan.TargetIP.ValueString())
	}
	return r.shared.Client.DMZOff()
}

func (r *dmzResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_dmz.")
		return
	}
	var plan dmzModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Enabled.ValueBool() && plan.TargetIP.ValueString() == "" {
		resp.Diagnostics.AddError("target_ip required", "target_ip must be set when enabled = true.")
		return
	}
	if err := r.apply(plan); err != nil {
		resp.Diagnostics.AddError("DMZ apply failed", err.Error())
		return
	}
	plan.ID = types.StringValue("singleton")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dmzResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state dmzModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := r.shared.Client.DMZ()
	if err != nil {
		resp.Diagnostics.AddError("DMZ read failed", err.Error())
		return
	}
	state.ID = types.StringValue("singleton")
	state.Enabled = types.BoolValue(toBool(raw["enable"]))
	state.TargetIP = types.StringValue(toStr(raw["ipaddress"]))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dmzResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_dmz.")
		return
	}
	var plan dmzModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Enabled.ValueBool() && plan.TargetIP.ValueString() == "" {
		resp.Diagnostics.AddError("target_ip required", "target_ip must be set when enabled = true.")
		return
	}
	if err := r.apply(plan); err != nil {
		resp.Diagnostics.AddError("DMZ apply failed", err.Error())
		return
	}
	plan.ID = types.StringValue("singleton")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dmzResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"bbox_dmz destroy is a no-op",
		"Removing this resource only removes it from Terraform state; the router's DMZ setting is unchanged.",
	)
}

func (r *dmzResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
