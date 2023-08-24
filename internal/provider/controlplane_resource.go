package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/eschercloudai/eckctl/pkg/generated"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &controlPlaneResource{}
	_ resource.ResourceWithConfigure = &controlPlaneResource{}
)

// NewControlPlaneResource is a helper function to simplify the provider implementation.
func NewControlPlaneResource() resource.Resource {
	return &controlPlaneResource{}
}

// controlPlaneResource is the resource implementation.
type controlPlaneResource struct {
	client *generated.ClientWithResponses
}

// Configure adds the provider configured client to the resource.
func (r *controlPlaneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*generated.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *generated.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *controlPlaneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_controlplane"
}

// Schema defines the schema for the resource.
func (r *controlPlaneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the ECK Control Plane.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"applicationbundle": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"version": schema.StringAttribute{
						Description: "The version of the ECK Control Plane.",
						Required:    true,
					},
					"autoupgrade": schema.BoolAttribute{
						Description: "Whether automatic upgrades of the ECK Control Plane are enabled.",
						Required:    true,
					},
				},
			},
		},
	}
}

// Create a new resource.
func (r *controlPlaneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan controlPlaneModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Match the default specified in the UI
	upgradeWindow := &generated.ApplicationBundleAutoUpgrade{
		DaysOfWeek: &generated.AutoUpgradeDaysOfWeek{
			Monday: &generated.TimeWindow{
				Start: 0,
				End:   7,
			},
			Tuesday: &generated.TimeWindow{
				Start: 0,
				End:   7,
			},
			Wednesday: &generated.TimeWindow{
				Start: 0,
				End:   7,
			},
			Thursday: &generated.TimeWindow{
				Start: 0,
				End:   7,
			},
			Friday: &generated.TimeWindow{
				Start: 0,
				End:   7,
			},
		},
	}

	// Generate API request body from plan
	controlplane := generated.ControlPlane{
		Name: plan.Name.ValueString(),
		ApplicationBundle: generated.ApplicationBundle{
			Name:    "control-plane-" + plan.ApplicationBundle.Version.ValueString(),
			Version: plan.ApplicationBundle.Version.ValueString(),
		},
		ApplicationBundleAutoUpgrade: upgradeWindow,
	}

	// Create new controlplane
	_, err := r.client.PostApiV1ControlplanesWithResponse(ctx, controlplane)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating controlplane",
			"Could not create controlplane, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan = controlPlaneModel{
		Name: types.StringValue(controlplane.Name),
		ApplicationBundle: applicationBundleModel{
			Version:     types.StringValue(controlplane.ApplicationBundle.Version),
			AutoUpgrade: types.BoolValue(IsDaysOfWeekSet(controlplane.ApplicationBundleAutoUpgrade)),
		},
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *controlPlaneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state controlPlaneModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed values from Unikorn
	controlplanes, err := r.client.GetApiV1ControlplanesControlPlaneName(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Control Plane information",
			"Could not read Control Plane ID "+state.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	body, err := io.ReadAll(controlplanes.Body)
	if err != nil {
		return
	}

	controlPlane := generated.ControlPlane{}
	err = json.Unmarshal(body, &controlPlane)
	if err != nil {
		fmt.Println(err)
	}

	// Overwrite items with refreshed state
	state = controlPlaneModel{
		Name: types.StringValue(controlPlane.Name),
		ApplicationBundle: applicationBundleModel{
			Version:     types.StringValue(controlPlane.ApplicationBundle.Version),
			AutoUpgrade: types.BoolValue(IsDaysOfWeekSet(controlPlane.ApplicationBundleAutoUpgrade)),
		},
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error", "Cannot set state"+err.Error(),
		)
		return
	}
}

func (r *controlPlaneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan controlPlaneModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state controlPlaneModel
	req.State.Get(ctx, &state)

	var u generated.ApplicationBundleAutoUpgrade

	// Generate API request body from plan
	controlplane := generated.ControlPlane{
		Name: plan.Name.ValueString(),
		ApplicationBundle: generated.ApplicationBundle{
			Name:    "control-plane-" + plan.ApplicationBundle.Version.ValueString(),
			Version: plan.ApplicationBundle.Version.String(),
		},
		ApplicationBundleAutoUpgrade: &u,
	}

	// Update controlplane
	h, err := r.client.PutApiV1ControlplanesControlPlaneNameWithResponse(ctx, state.Name.ValueString(), controlplane)
	if h.HTTPResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Error updating controlplane",
			"Received unexpected HTTP response: "+fmt.Sprintf("%v", h.HTTPResponse.StatusCode),
		)
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating controlplane",
			"Could not update controlplane, unexpected error: "+err.Error(),
		)
		return
	}

	// Get refreshed values from API
	controlplanes, err := r.client.GetApiV1ControlplanesControlPlaneName(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Control Plane information",
			"Could not read Control Plane ID "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	body, err := io.ReadAll(controlplanes.Body)
	if err != nil {
		return
	}

	controlPlane := generated.ControlPlane{}
	err = json.Unmarshal(body, &controlPlane)
	if err != nil {
		fmt.Println(err)
	}

	// Map response body to schema and populate Computed attribute values
	plan = controlPlaneModel{
		Name: types.StringValue(controlplane.Name),
		ApplicationBundle: applicationBundleModel{
			AutoUpgrade: types.BoolValue(IsDaysOfWeekSet(controlPlane.ApplicationBundleAutoUpgrade)),
			Version:     types.StringValue(controlplane.ApplicationBundle.Version),
		},
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *controlPlaneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state controlPlaneModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing control plane
	_, err := r.client.DeleteApiV1ControlplanesControlPlaneName(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Control Plane",
			"Could not delete control plane, unexpected error: "+err.Error(),
		)
		return
	}
}
