package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*wifiBandResource)(nil)
	_ resource.ResourceWithConfigure   = (*wifiBandResource)(nil)
	_ resource.ResourceWithImportState = (*wifiBandResource)(nil)
)

// NewWifiBandResource is the resource constructor exposed to the provider.
func NewWifiBandResource() resource.Resource { return &wifiBandResource{} }

type wifiBandResource struct {
	shared *SharedClient
}

type wifiBandModel struct {
	ID         types.String `tfsdk:"id"`
	Band       types.String `tfsdk:"band"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	SSID       types.String `tfsdk:"ssid"`
	Passphrase types.String `tfsdk:"passphrase"`
	Channel    types.Int64  `tfsdk:"channel"`
}

func (r *wifiBandResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wifi_band"
}

func (r *wifiBandResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "WiFi settings for one radio band. Singleton-per-band: destroy is a no-op (bands are permanent).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "Same as band."},
			"band": schema.StringAttribute{
				Required:    true,
				Description: "24, 5, or 6. Immutable.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled":    schema.BoolAttribute{Optional: true, Computed: true, Description: "Radio on/off."},
			"ssid":       schema.StringAttribute{Optional: true, Computed: true},
			"passphrase": schema.StringAttribute{Optional: true, Computed: true, Sensitive: true},
			"channel":    schema.Int64Attribute{Optional: true, Computed: true, Description: "0 = auto."},
		},
	}
}

func (r *wifiBandResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

func validateBand(b string) bool { return b == "24" || b == "5" || b == "6" }

// applyWifi pushes each configured attribute through its respective setter.
// Unset (null/unknown) attributes are left untouched.
func (r *wifiBandResource) applyWifi(m wifiBandModel) error {
	band := m.Band.ValueString()
	if !m.SSID.IsNull() && !m.SSID.IsUnknown() {
		if err := r.shared.Client.WifiSSIDSet(band, m.SSID.ValueString()); err != nil {
			return err
		}
	}
	if !m.Passphrase.IsNull() && !m.Passphrase.IsUnknown() {
		if err := r.shared.Client.WifiKeySet(band, m.Passphrase.ValueString()); err != nil {
			return err
		}
	}
	if !m.Channel.IsNull() && !m.Channel.IsUnknown() {
		ch := "auto"
		if v := m.Channel.ValueInt64(); v != 0 {
			ch = strconv.Itoa(int(v))
		}
		if err := r.shared.Client.WifiChannelSet(band, ch); err != nil {
			return err
		}
	}
	if !m.Enabled.IsNull() && !m.Enabled.IsUnknown() {
		if err := r.shared.Client.WifiBandToggle(band, m.Enabled.ValueBool()); err != nil {
			return err
		}
	}
	return nil
}

func (r *wifiBandResource) refresh(m *wifiBandModel) error {
	band := m.Band.ValueString()
	raw, err := r.shared.Client.WifiBand(band)
	if err != nil {
		return err
	}
	radio, _ := raw["radio"].(map[string]any)
	if radio == nil {
		radio = map[string]any{}
	}
	m.ID = types.StringValue(band)
	m.Enabled = types.BoolValue(toBool(radio["enable"]))
	m.SSID = types.StringValue(toStr(raw["ssid"]))
	// The wpaKey is not always returned; preserve any prior value rather than
	// silently blanking a sensitive attribute.
	if v := toStr(raw["wpaKey"]); v != "" {
		m.Passphrase = types.StringValue(v)
	} else if m.Passphrase.IsNull() || m.Passphrase.IsUnknown() {
		m.Passphrase = types.StringValue("")
	}
	m.Channel = types.Int64Value(int64(toInt(radio["channel"])))
	return nil
}

func (r *wifiBandResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_wifi_band.")
		return
	}
	var plan wifiBandModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !validateBand(plan.Band.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("band"), "Invalid band", "band must be 24, 5, or 6")
		return
	}
	if err := r.applyWifi(plan); err != nil {
		resp.Diagnostics.AddError("WiFi apply failed", err.Error())
		return
	}
	if err := r.refresh(&plan); err != nil {
		resp.Diagnostics.AddError("WiFi refresh failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wifiBandResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state wifiBandModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.refresh(&state); err != nil {
		resp.Diagnostics.AddError("WiFi refresh failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *wifiBandResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_wifi_band.")
		return
	}
	var state, plan wifiBandModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Only push setters for attributes that actually changed — bands are
	// singletons, so we cannot delete+recreate.
	band := plan.Band.ValueString()
	if plan.SSID.ValueString() != state.SSID.ValueString() && !plan.SSID.IsNull() && !plan.SSID.IsUnknown() {
		if err := r.shared.Client.WifiSSIDSet(band, plan.SSID.ValueString()); err != nil {
			resp.Diagnostics.AddError("WiFi SSID set failed", err.Error())
			return
		}
	}
	if plan.Passphrase.ValueString() != state.Passphrase.ValueString() && !plan.Passphrase.IsNull() && !plan.Passphrase.IsUnknown() {
		if err := r.shared.Client.WifiKeySet(band, plan.Passphrase.ValueString()); err != nil {
			resp.Diagnostics.AddError("WiFi key set failed", err.Error())
			return
		}
	}
	if plan.Channel.ValueInt64() != state.Channel.ValueInt64() && !plan.Channel.IsNull() && !plan.Channel.IsUnknown() {
		ch := "auto"
		if v := plan.Channel.ValueInt64(); v != 0 {
			ch = strconv.Itoa(int(v))
		}
		if err := r.shared.Client.WifiChannelSet(band, ch); err != nil {
			resp.Diagnostics.AddError("WiFi channel set failed", err.Error())
			return
		}
	}
	if plan.Enabled.ValueBool() != state.Enabled.ValueBool() && !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		if err := r.shared.Client.WifiBandToggle(band, plan.Enabled.ValueBool()); err != nil {
			resp.Diagnostics.AddError("WiFi band toggle failed", err.Error())
			return
		}
	}
	if err := r.refresh(&plan); err != nil {
		resp.Diagnostics.AddError("WiFi refresh failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wifiBandResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"bbox_wifi_band destroy is a no-op",
		"WiFi bands are permanent router features and cannot be destroyed. "+
			"To disable the radio, set `enabled = false`. "+
			"Removing this resource block only removes it from Terraform state.",
	)
}

func (r *wifiBandResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("band"), req, resp)
}
