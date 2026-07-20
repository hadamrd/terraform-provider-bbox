package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*voipBlockAnonResource)(nil)
	_ resource.ResourceWithConfigure   = (*voipBlockAnonResource)(nil)
	_ resource.ResourceWithImportState = (*voipBlockAnonResource)(nil)
)

// NewVoIPBlockAnonymousResource is the resource constructor exposed to the provider.
func NewVoIPBlockAnonymousResource() resource.Resource { return &voipBlockAnonResource{} }

type voipBlockAnonResource struct {
	shared *SharedClient
}

type voipBlockAnonModel struct {
	Line    types.Int64 `tfsdk:"line"`
	Blocked types.Bool  `tfsdk:"blocked"`
}

func (r *voipBlockAnonResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_voip_block_anonymous"
}

func (r *voipBlockAnonResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Blocks anonymous / caller-ID-withheld calls on a VoIP line — a simple anti-spam-call measure.",
		Attributes: map[string]schema.Attribute{
			"line": schema.Int64Attribute{
				Required:    true,
				Description: "Phone line number (1 or 2). Immutable.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"blocked": schema.BoolAttribute{Required: true, Description: "Whether anonymous calls are blocked."},
		},
	}
}

func (r *voipBlockAnonResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

func (r *voipBlockAnonResource) set(plan voipBlockAnonModel) error {
	return r.shared.Client.VoIPBlockAnonymous(int(plan.Line.ValueInt64()), plan.Blocked.ValueBool())
}

func (r *voipBlockAnonResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_voip_block_anonymous.")
		return
	}
	var plan voipBlockAnonModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.set(plan); err != nil {
		resp.Diagnostics.AddError("VoIP block-anonymous failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *voipBlockAnonResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state voipBlockAnonModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	blocked, err := r.shared.Client.VoIPAnonBlocked(int(state.Line.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("VoIP read failed", err.Error())
		return
	}
	state.Blocked = types.BoolValue(blocked)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *voipBlockAnonResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_voip_block_anonymous.")
		return
	}
	var plan voipBlockAnonModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.set(plan); err != nil {
		resp.Diagnostics.AddError("VoIP block-anonymous failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete restores the line to "anonymous allowed" (the default), since removing
// the resource means we no longer assert a block.
func (r *voipBlockAnonResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.shared == nil {
		return
	}
	var state voipBlockAnonModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.shared.Client.VoIPBlockAnonymous(int(state.Line.ValueInt64()), false); err != nil {
		resp.Diagnostics.AddError("VoIP unblock-on-destroy failed", err.Error())
	}
}

func (r *voipBlockAnonResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	line, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "expected the line number (1 or 2), got "+req.ID)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("line"), line)...)
}
