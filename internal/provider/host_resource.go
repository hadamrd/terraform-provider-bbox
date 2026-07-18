package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*hostResource)(nil)
	_ resource.ResourceWithConfigure   = (*hostResource)(nil)
	_ resource.ResourceWithImportState = (*hostResource)(nil)
)

// NewHostResource is the resource constructor exposed to the provider.
func NewHostResource() resource.Resource { return &hostResource{} }

type hostResource struct {
	shared *SharedClient
}

type hostResourceModel struct {
	ID       types.Int64  `tfsdk:"id"`
	MAC      types.String `tfsdk:"mac"`
	Hostname types.String `tfsdk:"hostname"`
	Blocked  types.Bool   `tfsdk:"blocked"`
}

func (r *hostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (r *hostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Mutable metadata (hostname, block state) for a LAN host identified by MAC. The host must already have connected once.",
		Attributes: map[string]schema.Attribute{
			"id":       schema.Int64Attribute{Computed: true, Description: "Router-assigned host ID."},
			"mac":      schema.StringAttribute{Required: true, Description: "MAC address (any case, dashes or colons)."},
			"hostname": schema.StringAttribute{Optional: true, Computed: true},
			"blocked":  schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
		},
	}
}

func (r *hostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

// lookupHost finds the host by MAC via HostBy. Returns id + current hostname + blocked.
func (r *hostResource) lookupHost(mac string) (int, string, bool, error) {
	h, err := r.shared.Client.HostBy(mac)
	if err != nil {
		return 0, "", false, err
	}
	return toInt(h["id"]), toStr(h["hostname"]), toBool(h["blocked"]), nil
}

func (r *hostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_host.")
		return
	}
	var plan hostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	mac := normaliseMAC(plan.MAC.ValueString())
	plan.MAC = types.StringValue(mac)

	id, curHostname, curBlocked, err := r.lookupHost(mac)
	if err != nil {
		resp.Diagnostics.AddError(
			"Host not found",
			"host with MAC "+mac+" not found on router — it must have connected at least once before it can be declared: "+err.Error(),
		)
		return
	}
	plan.ID = types.Int64Value(int64(id))

	if !plan.Hostname.IsNull() && !plan.Hostname.IsUnknown() && plan.Hostname.ValueString() != curHostname {
		if err := r.shared.Client.HostRename(id, plan.Hostname.ValueString()); err != nil {
			resp.Diagnostics.AddError("Host rename failed", err.Error())
			return
		}
	} else {
		plan.Hostname = types.StringValue(curHostname)
	}

	want := plan.Blocked.ValueBool()
	if want != curBlocked {
		var err error
		if want {
			err = r.shared.Client.HostBlock(id)
		} else {
			err = r.shared.Client.HostUnblock(id)
		}
		if err != nil {
			resp.Diagnostics.AddError("Host block/unblock failed", err.Error())
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state hostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	mac := normaliseMAC(state.MAC.ValueString())
	id, hostname, blocked, err := r.lookupHost(mac)
	if err != nil {
		// Host disappeared from the router.
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.Int64Value(int64(id))
	state.MAC = types.StringValue(mac)
	state.Hostname = types.StringValue(hostname)
	state.Blocked = types.BoolValue(blocked)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *hostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_host.")
		return
	}
	var state, plan hostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := int(state.ID.ValueInt64())
	if id == 0 {
		resp.Diagnostics.AddError("Missing host ID", "cannot update bbox_host without a router-assigned ID in state")
		return
	}
	if plan.Hostname.ValueString() != state.Hostname.ValueString() && !plan.Hostname.IsNull() && !plan.Hostname.IsUnknown() {
		if err := r.shared.Client.HostRename(id, plan.Hostname.ValueString()); err != nil {
			resp.Diagnostics.AddError("Host rename failed", err.Error())
			return
		}
	}
	if plan.Blocked.ValueBool() != state.Blocked.ValueBool() {
		var err error
		if plan.Blocked.ValueBool() {
			err = r.shared.Client.HostBlock(id)
		} else {
			err = r.shared.Client.HostUnblock(id)
		}
		if err != nil {
			resp.Diagnostics.AddError("Host block/unblock failed", err.Error())
			return
		}
	}
	plan.ID = state.ID
	plan.MAC = types.StringValue(normaliseMAC(plan.MAC.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"bbox_host destroy is a no-op",
		"Resource removed from state — the host record on the router is unchanged. "+
			"To rename or block a host, keep the resource.",
	)
}

func (r *hostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("mac"), req, resp)
}
