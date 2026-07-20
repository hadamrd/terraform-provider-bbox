package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*wifiGuestResource)(nil)
	_ resource.ResourceWithConfigure   = (*wifiGuestResource)(nil)
	_ resource.ResourceWithImportState = (*wifiGuestResource)(nil)
)

// NewWifiGuestResource is the resource constructor exposed to the provider.
func NewWifiGuestResource() resource.Resource { return &wifiGuestResource{} }

type wifiGuestResource struct {
	shared *SharedClient
}

type wifiGuestModel struct {
	ID      types.String `tfsdk:"id"`
	Enabled types.Bool   `tfsdk:"enabled"`
	SSID    types.String `tfsdk:"ssid"`
}

func (r *wifiGuestResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wifi_guest"
}

func (r *wifiGuestResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Singleton guest WiFi network (on/off + SSID). The passphrase is intentionally " +
			"unmanaged to avoid accidentally rotating it; set it via `bbox wifi guest key` if needed.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true, Description: "Always \"singleton\"."},
			"enabled": schema.BoolAttribute{Required: true, Description: "Whether guest WiFi is broadcasting."},
			"ssid":    schema.StringAttribute{Optional: true, Computed: true, Description: "Guest network SSID."},
		},
	}
}

func (r *wifiGuestResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

func (r *wifiGuestResource) apply(plan wifiGuestModel) error {
	if !plan.SSID.IsNull() && !plan.SSID.IsUnknown() && plan.SSID.ValueString() != "" {
		if err := r.shared.Client.GuestSSIDSet(plan.SSID.ValueString()); err != nil {
			return err
		}
	}
	return r.shared.Client.WifiGuestToggle(plan.Enabled.ValueBool())
}

func (r *wifiGuestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_wifi_guest.")
		return
	}
	var plan wifiGuestModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(plan); err != nil {
		resp.Diagnostics.AddError("Guest WiFi apply failed", err.Error())
		return
	}
	if err := r.refresh(&plan); err != nil {
		resp.Diagnostics.AddError("Guest WiFi refresh failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wifiGuestResource) refresh(m *wifiGuestModel) error {
	ge, err := r.shared.Client.GuestEnable()
	if err != nil {
		return err
	}
	m.ID = types.StringValue("singleton")
	m.Enabled = types.BoolValue(toBool(ge["enable"]))
	guest, err := r.shared.Client.WifiGuest()
	if err == nil {
		if g24, ok := guest["guest24"].(map[string]any); ok {
			m.SSID = types.StringValue(toStr(g24["SSID"]))
		}
	}
	if m.SSID.IsNull() || m.SSID.IsUnknown() {
		m.SSID = types.StringValue("")
	}
	return nil
}

func (r *wifiGuestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state wifiGuestModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.refresh(&state); err != nil {
		resp.Diagnostics.AddError("Guest WiFi read failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *wifiGuestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_wifi_guest.")
		return
	}
	var plan wifiGuestModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(plan); err != nil {
		resp.Diagnostics.AddError("Guest WiFi apply failed", err.Error())
		return
	}
	if err := r.refresh(&plan); err != nil {
		resp.Diagnostics.AddError("Guest WiFi refresh failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wifiGuestResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"bbox_wifi_guest destroy is a no-op",
		"Removing this resource only removes it from Terraform state; the guest network is unchanged.",
	)
}

func (r *wifiGuestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
