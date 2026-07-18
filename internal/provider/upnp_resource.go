package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*upnpResource)(nil)
	_ resource.ResourceWithConfigure   = (*upnpResource)(nil)
	_ resource.ResourceWithImportState = (*upnpResource)(nil)
)

// NewUPnPResource is the resource constructor exposed to the provider.
func NewUPnPResource() resource.Resource { return &upnpResource{} }

type upnpResource struct {
	shared *SharedClient
}

type upnpModel struct {
	ID      types.String `tfsdk:"id"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func (r *upnpResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_upnp"
}

func (r *upnpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Singleton for the UPnP IGD toggle.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true, Description: "Always \"singleton\"."},
			"enabled": schema.BoolAttribute{Required: true},
		},
	}
}

func (r *upnpResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

func (r *upnpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_upnp.")
		return
	}
	var plan upnpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.shared.Client.UPnPToggle(plan.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("UPnP toggle failed", err.Error())
		return
	}
	plan.ID = types.StringValue("singleton")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *upnpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state upnpModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := r.shared.Client.UPnP()
	if err != nil {
		resp.Diagnostics.AddError("UPnP read failed", err.Error())
		return
	}
	state.ID = types.StringValue("singleton")
	state.Enabled = types.BoolValue(toBool(raw["enable"]))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *upnpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_upnp.")
		return
	}
	var plan upnpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.shared.Client.UPnPToggle(plan.Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("UPnP toggle failed", err.Error())
		return
	}
	plan.ID = types.StringValue("singleton")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *upnpResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"bbox_upnp destroy is a no-op",
		"Removing this resource block only removes it from Terraform state; the router's UPnP flag is unchanged.",
	)
}

func (r *upnpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
