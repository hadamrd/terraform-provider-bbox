package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*usbResource)(nil)
	_ resource.ResourceWithConfigure   = (*usbResource)(nil)
	_ resource.ResourceWithImportState = (*usbResource)(nil)
)

// NewUSBResource is the resource constructor exposed to the provider.
func NewUSBResource() resource.Resource { return &usbResource{} }

type usbResource struct {
	shared *SharedClient
}

type usbModel struct {
	ID          types.String `tfsdk:"id"`
	USB3Enabled types.Bool   `tfsdk:"usb3_enabled"`
}

func (r *usbResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_usb"
}

func (r *usbResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Singleton USB settings. USB 3.0 mode can interfere with 2.4 GHz WiFi; disabling it " +
			"also trims attack surface when no USB device is attached.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Computed: true, Description: "Always \"singleton\"."},
			"usb3_enabled": schema.BoolAttribute{Required: true, Description: "Whether USB 3.0 mode is enabled."},
		},
	}
}

func (r *usbResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

func (r *usbResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_usb.")
		return
	}
	var plan usbModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.shared.Client.USB3Toggle(plan.USB3Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("USB3 toggle failed", err.Error())
		return
	}
	plan.ID = types.StringValue("singleton")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *usbResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state usbModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	on, err := r.shared.Client.USB3Enabled()
	if err != nil {
		resp.Diagnostics.AddError("USB read failed", err.Error())
		return
	}
	state.ID = types.StringValue("singleton")
	state.USB3Enabled = types.BoolValue(on)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *usbResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_usb.")
		return
	}
	var plan usbModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.shared.Client.USB3Toggle(plan.USB3Enabled.ValueBool()); err != nil {
		resp.Diagnostics.AddError("USB3 toggle failed", err.Error())
		return
	}
	plan.ID = types.StringValue("singleton")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *usbResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"bbox_usb destroy is a no-op",
		"Removing this resource only removes it from Terraform state; USB settings are unchanged.",
	)
}

func (r *usbResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
