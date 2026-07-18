package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*dhcpReservationResource)(nil)
	_ resource.ResourceWithConfigure   = (*dhcpReservationResource)(nil)
	_ resource.ResourceWithImportState = (*dhcpReservationResource)(nil)
)

// NewDHCPReservationResource is the resource constructor exposed to the provider.
func NewDHCPReservationResource() resource.Resource { return &dhcpReservationResource{} }

type dhcpReservationResource struct {
	shared *SharedClient
}

type dhcpReservationModel struct {
	ID        types.Int64  `tfsdk:"id"`
	MAC       types.String `tfsdk:"mac"`
	IPAddress types.String `tfsdk:"ip_address"`
	Hostname  types.String `tfsdk:"hostname"`
}

func (r *dhcpReservationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dhcp_reservation"
}

func (r *dhcpReservationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A DHCP static reservation on the Bbox.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "Router-assigned client ID.",
			},
			"mac": schema.StringAttribute{
				Required:    true,
				Description: "MAC address (any case, dashes or colons).",
			},
			"ip_address": schema.StringAttribute{
				Required:    true,
				Description: "IP to reserve.",
			},
			"hostname": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Optional hostname.",
			},
		},
	}
}

func (r *dhcpReservationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

func (r *dhcpReservationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_dhcp_reservation.")
		return
	}
	var plan dhcpReservationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	mac := normaliseMAC(plan.MAC.ValueString())
	plan.MAC = types.StringValue(mac)

	if err := r.shared.Client.DHCPReserve(mac, plan.IPAddress.ValueString(), plan.Hostname.ValueString()); err != nil {
		resp.Diagnostics.AddError("DHCP reserve failed", err.Error())
		return
	}

	id := r.findReservationID(mac, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.Int64Value(int64(id))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dhcpReservationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state dhcpReservationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	root, err := r.shared.Client.DHCPClients()
	if err != nil {
		resp.Diagnostics.AddError("DHCP clients list failed", err.Error())
		return
	}
	mac := normaliseMAC(state.MAC.ValueString())
	for _, m := range walkDHCPClients(root) {
		if normaliseMAC(toStr(m["macaddress"])) != mac {
			continue
		}
		state.ID = types.Int64Value(int64(toInt(m["id"])))
		state.IPAddress = types.StringValue(toStr(m["ipaddress"]))
		if hn := toStr(m["hostname"]); hn != "" {
			state.Hostname = types.StringValue(hn)
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *dhcpReservationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_dhcp_reservation.")
		return
	}
	var state dhcpReservationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var plan dhcpReservationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Delete+recreate — no PATCH available.
	if id := int(state.ID.ValueInt64()); id != 0 {
		if err := r.shared.Client.DHCPReservationDel(id); err != nil {
			resp.Diagnostics.AddError("DHCP delete-for-update failed", err.Error())
			return
		}
	}
	mac := normaliseMAC(plan.MAC.ValueString())
	plan.MAC = types.StringValue(mac)
	if err := r.shared.Client.DHCPReserve(mac, plan.IPAddress.ValueString(), plan.Hostname.ValueString()); err != nil {
		resp.Diagnostics.AddError("DHCP reserve failed", err.Error())
		return
	}
	id := r.findReservationID(mac, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.Int64Value(int64(id))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dhcpReservationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.shared == nil {
		return
	}
	var state dhcpReservationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := int(state.ID.ValueInt64())
	if id == 0 {
		return
	}
	if err := r.shared.Client.DHCPReservationDel(id); err != nil {
		resp.Diagnostics.AddError("DHCP reservation delete failed", err.Error())
	}
}

func (r *dhcpReservationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by MAC — the ID is router-assigned so ImportStateVerify won't line up.
	resource.ImportStatePassthroughID(ctx, path.Root("mac"), req, resp)
}

// findReservationID looks up the router-assigned client ID by MAC after a
// reserve call. Diagnostics are appended to the caller's slice.
func (r *dhcpReservationResource) findReservationID(mac string, diags *diag.Diagnostics) int {
	root, err := r.shared.Client.DHCPClients()
	if err != nil {
		diags.AddError("DHCP clients list failed", err.Error())
		return 0
	}
	for _, m := range walkDHCPClients(root) {
		if normaliseMAC(toStr(m["macaddress"])) == mac {
			return toInt(m["id"])
		}
	}
	diags.AddWarning(
		"Reservation created but ID not yet visible",
		"The router hasn't surfaced the new client entry for MAC "+mac+" yet; state ID left at 0.",
	)
	return 0
}
