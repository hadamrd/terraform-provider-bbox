package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*parentalControlResource)(nil)
	_ resource.ResourceWithConfigure   = (*parentalControlResource)(nil)
	_ resource.ResourceWithImportState = (*parentalControlResource)(nil)
)

// NewParentalControlResource is the resource constructor exposed to the provider.
func NewParentalControlResource() resource.Resource { return &parentalControlResource{} }

type parentalControlResource struct {
	shared *SharedClient
}

type parentalControlModel struct {
	ID            types.String `tfsdk:"id"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	DefaultPolicy types.String `tfsdk:"default_policy"`
}

func (r *parentalControlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parental_control"
}

func (r *parentalControlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Singleton for parental-control master settings. Access windows are managed " +
			"separately via bbox_parental_rule, and devices are enrolled by MAC via the CLI.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true, Description: "Always \"singleton\"."},
			"enabled": schema.BoolAttribute{Required: true, Description: "Whether parental-control scheduling is active."},
			"default_policy": schema.StringAttribute{
				Optional: true, Computed: true,
				Default:     stringdefault.StaticString("Forbidden"),
				Description: "Access outside configured windows: \"Forbidden\" (windows grant access) or \"Accept\" (windows restrict).",
				Validators:  []validator.String{stringvalidator.OneOf("Forbidden", "Accept")},
			},
		},
	}
}

func (r *parentalControlResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

// push applies enabled + default_policy to the router.
func (r *parentalControlResource) push(plan parentalControlModel) error {
	if !plan.DefaultPolicy.IsNull() && !plan.DefaultPolicy.IsUnknown() {
		if err := r.shared.Client.ParentalSetPolicy(plan.DefaultPolicy.ValueString()); err != nil {
			return err
		}
	}
	return r.shared.Client.ParentalToggle(plan.Enabled.ValueBool())
}

func (r *parentalControlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_parental_control.")
		return
	}
	var plan parentalControlModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.push(plan); err != nil {
		resp.Diagnostics.AddError("Parental control apply failed", err.Error())
		return
	}
	plan.ID = types.StringValue("singleton")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *parentalControlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state parentalControlModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := r.shared.Client.Parental()
	if err != nil {
		resp.Diagnostics.AddError("Parental control read failed", err.Error())
		return
	}
	sched, _ := raw["scheduler"].(map[string]any)
	if sched == nil {
		sched = map[string]any{}
	}
	state.ID = types.StringValue("singleton")
	state.Enabled = types.BoolValue(toBool(sched["enable"]))
	if p := toStr(sched["defaultpolicy"]); p != "" {
		state.DefaultPolicy = types.StringValue(p)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *parentalControlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_parental_control.")
		return
	}
	var plan parentalControlModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.push(plan); err != nil {
		resp.Diagnostics.AddError("Parental control apply failed", err.Error())
		return
	}
	plan.ID = types.StringValue("singleton")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *parentalControlResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"bbox_parental_control destroy is a no-op",
		"Removing this resource only removes it from Terraform state; the router's parental-control settings are unchanged.",
	)
}

func (r *parentalControlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
