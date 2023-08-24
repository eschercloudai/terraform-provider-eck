package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/eschercloudai/eckctl/pkg/generated"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &clusterResource{}
	_ resource.ResourceWithConfigure = &clusterResource{}
)

// NewClusterResource is a helper function to simplify the provider implementation.
func NewClusterResource() resource.Resource {
	return &clusterResource{}
}

// clusterResource is the resource implementation.
type clusterResource struct {
	client *generated.ClientWithResponses
}

// Configure adds the provider configured client to the resource.
func (r *clusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *clusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

// Schema defines the schema for the resource.
func (r *clusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"eckcp": schema.StringAttribute{
				Required: true,
			},
			"applicationbundle": schema.StringAttribute{
				Required: true,
			},
			"kubeconfig": schema.StringAttribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"controlplane": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"disk": schema.Int64Attribute{
						Optional: true,
					},
					"flavor": schema.StringAttribute{
						Required: true,
					},
					"image": schema.StringAttribute{
						Required: true,
					},
					"replicas": schema.Int64Attribute{
						Required: true,
					},
					"version": schema.StringAttribute{
						Required: true,
					},
				},
			},
			"clusternetwork": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"dnsnameservers": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"nodeprefix": schema.StringAttribute{
						Optional: true,
					},
					"podprefix": schema.StringAttribute{
						Optional: true,
					},
					"serviceprefix": schema.StringAttribute{
						Optional: true,
					},
				},
			},
			"clusteropenstack": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"computeaz": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("nova"),
					},
					"externalnetworkid": schema.StringAttribute{
						Optional: true,
					},
					"sshkey": schema.StringAttribute{
						Optional: true,
					},
					"volumeaz": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("nova"),
					},
				},
			},
			"clusterfeatures": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"autoscaling": schema.BoolAttribute{
						Optional: true,
						Computed: true,
						Default:  booldefault.StaticBool(false),
					},
				},
			},
			"workloadnodepools": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"flavor": schema.StringAttribute{
							Required: true,
						},
						"image": schema.StringAttribute{
							Required: true,
						},
						"replicas": schema.Int64Attribute{
							Required: true,
						},
						"version": schema.StringAttribute{
							Optional: true,
						},
						"autoscaling": schema.SingleNestedAttribute{
							Optional: true,

							Attributes: map[string]schema.Attribute{
								"minimum": schema.Int64Attribute{
									Required: true,
								},
								"maximum": schema.Int64Attribute{
									Required: true,
								},
							},
						},
					},
				},
			},
		},
	}
}

// Create a new resource.
func (r *clusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "🦄 Create")
	// Retrieve values from plan
	var plan clusterModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cluster := generateKubernetesCluster(ctx, plan)

	// Create new cluster
	_, err := r.client.PostApiV1ControlplanesControlPlaneNameClusters(ctx, plan.EckCp.ValueString(), cluster)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating cluster",
			"Could not create cluster, unexpected error: "+err.Error(),
		)
		return
	}

	var kubeconfig string
	if cluster.Status.Status == "Provisioned" {
		kubeconfig = getKubeconfig(*r.client, ctx, plan.EckCp.ValueString(), cluster.Name)
	} else {
		kubeconfig = ""
	}

	// Refresh cluster details
	plan = generateClusterModel(ctx, cluster, plan.EckCp.ValueString(), kubeconfig)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "🦄 Read")
	// Get current state
	var state clusterModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed values from Unikorn
	kubernetesCluster, err := r.client.GetApiV1ControlplanesControlPlaneNameClustersClusterName(ctx, state.EckCp.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading cluster information",
			"Could not read cluster "+state.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	body, err := io.ReadAll(kubernetesCluster.Body)
	if err != nil {
		return
	}

	cluster := generated.KubernetesCluster{}
	err = json.Unmarshal(body, &cluster)
	if err != nil {
		fmt.Println(err)
	}

	var kubeconfig string
	if cluster.Status.Status == "Provisioned" {
		kubeconfig = getKubeconfig(*r.client, ctx, state.EckCp.ValueString(), cluster.Name)
	} else {
		kubeconfig = ""
	}

	// Refresh cluster details
	// Overwrite items with refreshed state
	state = generateClusterModel(ctx, cluster, state.EckCp.ValueString(), kubeconfig)

	var controlPlane controlPlaneNodesModel
	if state.ControlPlane.Disk != types.Int64Value(0) {
		controlPlane.Disk = state.ControlPlane.Disk
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

func (r *clusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "🦄 Update")
	// Retrieve values from plan
	var plan clusterModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	cluster := generateKubernetesCluster(ctx, plan)

	// Create new cluster
	_, err := r.client.PutApiV1ControlplanesControlPlaneNameClustersClusterName(ctx, "tftest", plan.Name.ValueString(), cluster)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating cluster",
			"Could not create cluster, unexpected error: "+err.Error(),
		)
		return
	}

	var kubeconfig string
	if cluster.Status.Status == "Provisioned" {
		kubeconfig = getKubeconfig(*r.client, ctx, plan.EckCp.ValueString(), cluster.Name)
	} else {
		kubeconfig = ""
	}

	// Refresh cluster details
	plan = generateClusterModel(ctx, cluster, plan.EckCp.ValueString(), kubeconfig)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *clusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "🦄 Delete")
	// Retrieve values from state
	var state clusterModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete cluster
	_, err := r.client.DeleteApiV1ControlplanesControlPlaneNameClustersClusterName(ctx, state.EckCp.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting cluster",
			"Could not delete cluster, unexpected error: "+err.Error(),
		)
		return
	}
}
