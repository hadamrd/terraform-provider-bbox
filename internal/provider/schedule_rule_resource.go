package provider

import (
	"context"

	"github.com/hadamrd/bbox-cli/pkg/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// scheduleRuleResource is shared by bbox_wifi_schedule and bbox_parental_rule —
// the WiFi-pause and parental-control schedulers use an identical rule shape
// ({name, occurency, intervals, enable}). The two differ only in the SDK calls,
// injected as closures by the constructor.
type scheduleRuleResource struct {
	shared   *SharedClient
	typeName string // e.g. "wifi_schedule" / "parental_rule"
	descr    string
	add      func(*client.Client, client.SchedulerRuleArgs) (int, error)
	del      func(*client.Client, int) error
	saved    func(*client.Client) ([]any, error) // returns the editable savedRules
}

var (
	_ resource.Resource                = (*scheduleRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*scheduleRuleResource)(nil)
	_ resource.ResourceWithImportState = (*scheduleRuleResource)(nil)
)

// NewWifiScheduleResource manages a recurring WiFi-pause window.
func NewWifiScheduleResource() resource.Resource {
	return &scheduleRuleResource{
		typeName: "wifi_schedule",
		descr: "A recurring WiFi-pause window: the WiFi radios switch off during the window " +
			"on the selected days. Adding an enabled window also turns the WiFi scheduler on.",
		add:   func(c *client.Client, a client.SchedulerRuleArgs) (int, error) { return c.WifiSchedulerAddRule(a) },
		del:   func(c *client.Client, id int) error { return c.WifiSchedulerDelRule(id) },
		saved: func(c *client.Client) ([]any, error) { _, s, err := c.WifiSchedulerRules(); return s, err },
	}
}

// NewParentalRuleResource manages a recurring parental-control access window.
func NewParentalRuleResource() resource.Resource {
	return &scheduleRuleResource{
		typeName: "parental_rule",
		descr: "A recurring parental-control access window. Interpreted against the parental " +
			"default policy (see bbox_parental_control): with policy \"Forbidden\", windows grant access.",
		add:   func(c *client.Client, a client.SchedulerRuleArgs) (int, error) { return c.ParentalAddRule(a) },
		del:   func(c *client.Client, id int) error { return c.ParentalDelRule(id) },
		saved: func(c *client.Client) ([]any, error) { _, s, err := c.ParentalRules(); return s, err },
	}
}

type scheduleRuleModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Days      types.List   `tfsdk:"days"`
	StartTime types.String `tfsdk:"start_time"`
	EndTime   types.String `tfsdk:"end_time"`
	Enabled   types.Bool   `tfsdk:"enabled"`
}

func (r *scheduleRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.typeName
}

func (r *scheduleRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: r.descr,
		Attributes: map[string]schema.Attribute{
			"id":   schema.Int64Attribute{Computed: true, Description: "Router-assigned window ID."},
			"name": schema.StringAttribute{Required: true, Description: "Window label (unique per scheduler)."},
			"days": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Days the window applies: any of mon,tue,wed,thu,fri,sat,sun.",
			},
			"start_time": schema.StringAttribute{Required: true, Description: "Start time HH:MM (24h)."},
			"end_time":   schema.StringAttribute{Required: true, Description: "End time HH:MM (24h)."},
			"enabled": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Whether the window is active.",
			},
		},
	}
}

func (r *scheduleRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	s, err := sharedFromAny(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider misconfigured", err.Error())
		return
	}
	r.shared = s
}

// argsFromModel converts a plan into SchedulerRuleArgs, validating days.
func (r *scheduleRuleResource) argsFromModel(ctx context.Context, m scheduleRuleModel) (client.SchedulerRuleArgs, error) {
	var days []string
	m.Days.ElementsAs(ctx, &days, false)
	occ, err := daysToOccurency(days)
	if err != nil {
		return client.SchedulerRuleArgs{}, err
	}
	return client.SchedulerRuleArgs{
		Name:      m.Name.ValueString(),
		Occurency: occ,
		Intervals: m.StartTime.ValueString() + "," + m.EndTime.ValueString(),
		Enable:    m.Enabled.ValueBool(),
	}, nil
}

func (r *scheduleRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot create bbox_"+r.typeName+".")
		return
	}
	var plan scheduleRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	args, err := r.argsFromModel(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid schedule", err.Error())
		return
	}
	id, err := r.add(r.shared.Client, args)
	if err != nil {
		resp.Diagnostics.AddError("Schedule add failed", err.Error())
		return
	}
	plan.ID = types.Int64Value(int64(id))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *scheduleRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.shared == nil {
		return
	}
	var state scheduleRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	saved, err := r.saved(r.shared.Client)
	if err != nil {
		resp.Diagnostics.AddError("Schedule list failed", err.Error())
		return
	}
	wanted := int(state.ID.ValueInt64())
	for _, rAny := range saved {
		m, _ := rAny.(map[string]any)
		if toInt(m["id"]) != wanted {
			continue
		}
		state.Name = types.StringValue(toStr(m["name"]))
		start, end := splitIntervals(toStr(m["intervals"]))
		state.StartTime = types.StringValue(start)
		state.EndTime = types.StringValue(end)
		state.Enabled = types.BoolValue(toBool(m["enable"]))
		days, diags := types.ListValueFrom(ctx, types.StringType, occurencyToDays(toStr(m["occurency"])))
		resp.Diagnostics.Append(diags...)
		state.Days = days
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	// drift: window vanished on the router.
	resp.State.RemoveResource(ctx)
}

func (r *scheduleRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.shared == nil {
		resp.Diagnostics.AddError("Provider not configured", "Cannot update bbox_"+r.typeName+".")
		return
	}
	var state, plan scheduleRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	args, err := r.argsFromModel(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid schedule", err.Error())
		return
	}
	// No in-place rule update endpoint; delete + recreate.
	if id := int(state.ID.ValueInt64()); id != 0 {
		if err := r.del(r.shared.Client, id); err != nil {
			resp.Diagnostics.AddError("Schedule delete-for-update failed", err.Error())
			return
		}
	}
	id, err := r.add(r.shared.Client, args)
	if err != nil {
		resp.Diagnostics.AddError("Schedule recreate failed", err.Error())
		return
	}
	plan.ID = types.Int64Value(int64(id))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *scheduleRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.shared == nil {
		return
	}
	var state scheduleRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if id := int(state.ID.ValueInt64()); id != 0 {
		if err := r.del(r.shared.Client, id); err != nil {
			resp.Diagnostics.AddError("Schedule delete failed", err.Error())
		}
	}
}

func (r *scheduleRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importInt64ID(ctx, req, resp)
}
