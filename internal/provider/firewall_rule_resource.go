package provider

import (
	"context"

	"github.com/hadamrd/bbox-cli/pkg/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*firewallRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*firewallRuleResource)(nil)
	_ resource.ResourceWithImportState = (*firewallRuleResource)(nil)
)

// NewFirewallRuleResource is the resource constructor exposed to the provider.
func NewFirewallRuleResource() resource.Resource { return &firewallRuleResource{} }

type firewallRuleResource struct {
	shared *SharedClient
}

type firewallRuleModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Action    types.String `tfsdk:"action"`
	Protocol  types.String `tfsdk:"protocol"`
	DstIP     types.String `tfsdk:"dst_ip"`
	DstPort   types.String `tfsdk:"dst_port"`
	SrcIP     types.String `tfsdk:"src_ip"`
	SrcPort   types.String `tfsdk:"src_port"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	IPVersion types.String `tfsdk:"ip_version"`
}

func (r *firewallRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_rule"
}

func (r *firewallRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A firewall rule on the Bbox.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.Int64Attribute{Computed: true, Description: "Router-assigned rule ID."},
			"name": schema.StringAttribute{Required: true, Description: "Rule description."},
			"action": schema.StringAttribute{
				Required:    true,
				Description: "Drop or Accept.",
				Validators: []validator.String{
					stringvalidator.OneOf("Drop", "Accept"),
				},
			},
			"protocol": schema.StringAttribute{
				Required:    true,
				Description: "tcp/udp/icmp/esp/ah/icmpv6/igmp/gre.",
				Validators: []validator.String{
					stringvalidator.OneOf("tcp", "udp", "icmp", "esp", "ah", "icmpv6", "igmp", "gre"),
				},
			},
			"dst_ip":   schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"dst_port": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(""), Description: "Port or range like 8000-8100."},
			"src_ip":   schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"src_port": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"enabled":  schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"ip_version": schema.StringAttribute{
				Optional: true, Computed: true,
				Default:     stringdefault.StaticString("IPv4"),
				Description: "IPv4 or IPv6.",
				Validators: []validator.String{
					stringvalidator.OneOf("IPv4", "IPv6"),
				},
			},
		},
	}
}

func (r *firewallRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

func firewallArgsFromModel(m firewallRuleModel) client.FirewallRuleArgs {
	return client.FirewallRuleArgs{
		Description: m.Name.ValueString(),
		Action:      m.Action.ValueString(),
		Protocol:    m.Protocol.ValueString(),
		DstIP:       m.DstIP.ValueString(),
		DstPort:     m.DstPort.ValueString(),
		SrcIP:       m.SrcIP.ValueString(),
		SrcPort:     m.SrcPort.ValueString(),
		IPVersion:   m.IPVersion.ValueString(),
		Enable:      m.Enabled.ValueBool(),
	}
}

func (r *firewallRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_firewall_rule.")
		return
	}
	var plan firewallRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	newID, err := r.shared.Client.FirewallRuleAdd(firewallArgsFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Firewall rule add failed", err.Error())
		return
	}
	plan.ID = types.Int64Value(int64(newID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state firewallRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rules, err := r.shared.Client.FirewallRules()
	if err != nil {
		resp.Diagnostics.AddError("Firewall rules list failed", err.Error())
		return
	}
	wanted := int(state.ID.ValueInt64())
	for _, rAny := range rules {
		m, _ := rAny.(map[string]any)
		if toInt(m["id"]) != wanted {
			continue
		}
		state.Name = types.StringValue(toStr(m["description"]))
		state.Action = types.StringValue(toStr(m["action"]))
		state.Protocol = types.StringValue(toStr(m["protocol"]))
		state.DstIP = types.StringValue(toStr(m["dstip"]))
		state.DstPort = types.StringValue(toStr(m["dstport"]))
		state.SrcIP = types.StringValue(toStr(m["srcip"]))
		state.SrcPort = types.StringValue(toStr(m["srcport"]))
		state.Enabled = types.BoolValue(toBool(m["enable"]))
		if v := toStr(m["ipprotocol"]); v != "" {
			state.IPVersion = types.StringValue(v)
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	// drift: rule vanished on the router.
	resp.State.RemoveResource(ctx)
}

func (r *firewallRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_firewall_rule.")
		return
	}
	var state, plan firewallRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If ONLY 'enabled' differs, use the in-place toggle. Otherwise delete+recreate.
	if firewallOnlyEnabledChanged(state, plan) {
		id := int(state.ID.ValueInt64())
		if err := r.shared.Client.FirewallRuleToggle(id, plan.Enabled.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Firewall rule toggle failed", err.Error())
			return
		}
		plan.ID = state.ID
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	if id := int(state.ID.ValueInt64()); id != 0 {
		if err := r.shared.Client.FirewallRuleDel(id); err != nil {
			resp.Diagnostics.AddError("Firewall rule delete-for-update failed", err.Error())
			return
		}
	}
	newID, err := r.shared.Client.FirewallRuleAdd(firewallArgsFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Firewall rule recreate failed", err.Error())
		return
	}
	plan.ID = types.Int64Value(int64(newID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.shared == nil {
		return
	}
	var state firewallRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := int(state.ID.ValueInt64())
	if id == 0 {
		return
	}
	if err := r.shared.Client.FirewallRuleDel(id); err != nil {
		resp.Diagnostics.AddError("Firewall rule delete failed", err.Error())
	}
}

func (r *firewallRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// firewallOnlyEnabledChanged returns true when the only differing attribute
// between state and plan is `enabled`.
func firewallOnlyEnabledChanged(state, plan firewallRuleModel) bool {
	if state.Enabled.ValueBool() == plan.Enabled.ValueBool() {
		return false
	}
	return state.Name.ValueString() == plan.Name.ValueString() &&
		state.Action.ValueString() == plan.Action.ValueString() &&
		state.Protocol.ValueString() == plan.Protocol.ValueString() &&
		state.DstIP.ValueString() == plan.DstIP.ValueString() &&
		state.DstPort.ValueString() == plan.DstPort.ValueString() &&
		state.SrcIP.ValueString() == plan.SrcIP.ValueString() &&
		state.SrcPort.ValueString() == plan.SrcPort.ValueString() &&
		state.IPVersion.ValueString() == plan.IPVersion.ValueString()
}
