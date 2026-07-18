package provider

import (
	"context"
	"fmt"

	"github.com/hadamrd/bbox-cli/pkg/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*natRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*natRuleResource)(nil)
	_ resource.ResourceWithImportState = (*natRuleResource)(nil)
)

// NewNATRuleResource is the resource constructor exposed to the provider.
func NewNATRuleResource() resource.Resource { return &natRuleResource{} }

type natRuleResource struct {
	shared *SharedClient
}

type natRuleModel struct {
	ID            types.Int64  `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	ExternalPort  types.Int64  `tfsdk:"external_port"`
	TargetIP      types.String `tfsdk:"target_ip"`
	InternalPort  types.Int64  `tfsdk:"internal_port"`
	Protocol      types.String `tfsdk:"protocol"`
	RemoteIP      types.String `tfsdk:"remote_ip"`
	SkipPortCheck types.Bool   `tfsdk:"skip_port_check"`
}

func (r *natRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nat_rule"
}

func (r *natRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A NAT / port-forward rule on the Bbox.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:      true,
				Description:   "Router-assigned numeric rule ID.",
				PlanModifiers: []planmodifier.Int64{
					// keep across updates until delete+recreate replaces it
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable description; used as the primary logical identifier.",
			},
			"external_port": schema.Int64Attribute{
				Required:    true,
				Description: "WAN port to expose. Must be inside the MAP-T range unless skip_port_check is true.",
			},
			"target_ip": schema.StringAttribute{
				Required:    true,
				Description: "LAN IP the traffic is forwarded to.",
			},
			"internal_port": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "LAN port. Defaults to external_port.",
			},
			"protocol": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("tcp"),
				Description: "tcp or udp.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"remote_ip": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Restrict source IP.",
			},
			"skip_port_check": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Bypass MAP-T port-range validation.",
			},
		},
	}
}

func (r *natRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

func (r *natRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_nat_rule without configured provider.")
		return
	}
	var plan natRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	proto := plan.Protocol.ValueString()
	if proto != "tcp" && proto != "udp" {
		resp.Diagnostics.AddAttributeError(path.Root("protocol"), "Invalid protocol", "must be tcp or udp")
		return
	}

	extPort := int(plan.ExternalPort.ValueInt64())
	if !plan.SkipPortCheck.ValueBool() {
		if err := checkMAPTRange(r.shared.Client, extPort); err != nil {
			resp.Diagnostics.AddError("MAP-T range check failed", err.Error())
			return
		}
	}

	intPort := int(plan.InternalPort.ValueInt64())
	if plan.InternalPort.IsNull() || plan.InternalPort.IsUnknown() || intPort == 0 {
		intPort = extPort
	}

	newID, err := r.shared.Client.NATAdd(client.NATAddArgs{
		Description:  plan.Name.ValueString(),
		ExternalPort: extPort,
		InternalIP:   plan.TargetIP.ValueString(),
		InternalPort: intPort,
		Protocol:     proto,
		RemoteIP:     plan.RemoteIP.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("NAT add failed", err.Error())
		return
	}

	plan.ID = types.Int64Value(int64(newID))
	plan.InternalPort = types.Int64Value(int64(intPort))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *natRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state natRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, err := r.shared.Client.NATRules()
	if err != nil {
		resp.Diagnostics.AddError("NAT list failed", err.Error())
		return
	}
	wanted := int(state.ID.ValueInt64())
	for _, rAny := range rules {
		m, _ := rAny.(map[string]any)
		if toInt(m["id"]) != wanted {
			continue
		}
		state.Name = types.StringValue(toStr(m["description"]))
		state.ExternalPort = types.Int64Value(int64(toInt(m["externalport"])))
		state.InternalPort = types.Int64Value(int64(toInt(m["internalport"])))
		state.TargetIP = types.StringValue(toStr(m["internalip"]))
		state.Protocol = types.StringValue(toStr(m["protocol"]))
		state.RemoteIP = types.StringValue(toStr(m["ipremote"]))
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	// drift: rule vanished on the router — remove from state so TF recreates.
	resp.State.RemoveResource(ctx)
}

func (r *natRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_nat_rule.")
		return
	}
	// Router has no PATCH: delete + recreate.
	var state natRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var plan natRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if id := int(state.ID.ValueInt64()); id != 0 {
		if err := r.shared.Client.NATDel(id); err != nil {
			resp.Diagnostics.AddError("NAT delete-for-update failed", err.Error())
			return
		}
	}

	proto := plan.Protocol.ValueString()
	if proto != "tcp" && proto != "udp" {
		resp.Diagnostics.AddAttributeError(path.Root("protocol"), "Invalid protocol", "must be tcp or udp")
		return
	}
	extPort := int(plan.ExternalPort.ValueInt64())
	if !plan.SkipPortCheck.ValueBool() {
		if err := checkMAPTRange(r.shared.Client, extPort); err != nil {
			resp.Diagnostics.AddError("MAP-T range check failed", err.Error())
			return
		}
	}
	intPort := int(plan.InternalPort.ValueInt64())
	if plan.InternalPort.IsNull() || plan.InternalPort.IsUnknown() || intPort == 0 {
		intPort = extPort
	}
	newID, err := r.shared.Client.NATAdd(client.NATAddArgs{
		Description:  plan.Name.ValueString(),
		ExternalPort: extPort,
		InternalIP:   plan.TargetIP.ValueString(),
		InternalPort: intPort,
		Protocol:     proto,
		RemoteIP:     plan.RemoteIP.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("NAT recreate failed", err.Error())
		return
	}
	plan.ID = types.Int64Value(int64(newID))
	plan.InternalPort = types.Int64Value(int64(intPort))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *natRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.shared == nil {
		return
	}
	var state natRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := int(state.ID.ValueInt64())
	if id == 0 {
		return
	}
	if err := r.shared.Client.NATDel(id); err != nil {
		resp.Diagnostics.AddError("NAT delete failed", err.Error())
	}
}

func (r *natRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// checkMAPTRange mirrors the CLI's port-range guard. Returns nil when the
// firmware reports no MAP-T range (native IPv4 or unparseable).
func checkMAPTRange(c *client.Client, externalPort int) error {
	wan, err := c.WAN()
	if err != nil {
		return err
	}
	ip, _ := wan["ip"].(map[string]any)
	if ip == nil {
		return nil
	}
	rng, _ := ip["portrange"].(string)
	lo, hi := parsePortRange(rng)
	if lo == 0 && hi == 0 {
		return nil
	}
	if externalPort < lo || externalPort > hi {
		return fmt.Errorf("port %d outside MAP-T range %d-%d; set skip_port_check=true to override", externalPort, lo, hi)
	}
	return nil
}
