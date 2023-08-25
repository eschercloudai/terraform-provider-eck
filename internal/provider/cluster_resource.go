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
				Description: "The name of the ECK cluster.",
				Required:    true,
			},
			"eckcp": schema.StringAttribute{
				Description: "The associated ECK Control Plane for the cluster.",
				Required:    true,
			},
			"applicationbundle": schema.StringAttribute{
				Description: "The version of the bundled components in the cluster.  See https://docs.eschercloud.ai/Kubernetes/Reference/compatibility_matrix for details.",
				Required:    true,
			},
			"kubeconfig": schema.StringAttribute{
				Description: "The kubeconfig for the cluster.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The provisioning status of the cluster.",
				Computed:    true,
			},
			"controlplane": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"disk": schema.Int64Attribute{
						Description: "Whether to use a dedicated persistent volume for control plane nodes. It is recommended to leave this unchecked, as ephemeral storage provides higher performance for Kubernetes' etcd database. If left unset, the default ephemeral storage size of 20GB is used.",
						Optional:    true,
					},
					"flavor": schema.StringAttribute{
						Description: "The flavor (size) of the machine.",
						Required:    true,
					},
					"image": schema.StringAttribute{
						Description: "Which OS image to use.  Must be a verified and signed ECK image",
						Required:    true,
					},
					"replicas": schema.Int64Attribute{
						Description: "How many replicas to provision in a control plane.  Must be an odd number, 3 is recommended.",
						Required:    true,
					},
					"version": schema.StringAttribute{
						Description: "The version of Kubernetes.  Must match the version bundled with the OS image.",
						Required:    true,
					},
				},
			},
			"clusternetwork": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"dnsnameservers": schema.ListAttribute{
						Description: "A list of DNS nameservers used by the OS.",
						ElementType: types.StringType,
						Optional:    true,
					},
					"nodeprefix": schema.StringAttribute{
						Description: "The CIDR-formatted IP address range to be used by Nodes in the cluster.",
						Optional:    true,
					},
					"podprefix": schema.StringAttribute{
						Description: "The CIDR-formatted IP address range to be used by Pods in the cluster.",
						Optional:    true,
					},
					"serviceprefix": schema.StringAttribute{
						Description: "The CIDR-formatted IP address range to be used by Services in the cluster.",
						Optional:    true,
					},
				},
			},
			"clusteropenstack": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"computeaz": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("nova"),
						Description: "OpenStack Compute Availability Zone. Defaults to `nova`.",
					},
					"externalnetworkid": schema.StringAttribute{
						Description: "UUID of the external network.",
						Optional:    true,
					},
					"sshkey": schema.StringAttribute{
						Description: "SSH key associated with the instance.",
						Optional:    true,
					},
					"volumeaz": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("nova"),
						Description: "OpenStack Cinder Availability Zone. Defaults to `nova`.",
					},
				},
			},
			"clusterfeatures": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Extra features allowing management of additional Kubernetes features that are considered standard.",
				Attributes: map[string]schema.Attribute{
					"autoscaling": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Enables Cluster Autoscaler, required for autoscaling workload pools.",
					},
					"ingress": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Whether to deploy the NGINX Ingress Controller.",
					},
					"longhorn": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Whether to enable Longhorn for persistent storage, which includes support for RWX.",
					},
					"prometheus": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Whether to enable the Prometheus Operator for monitoring.",
					},
				},
			},
			"workloadnodepools": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Name of the workload pool.",
							Required:    true,
						},
						"flavor": schema.StringAttribute{
							Description: "OpenStack flavor (size) for nodes in this pool.",
							Required:    true,
						},
						"image": schema.StringAttribute{
							Description: "Operating system image to use.  Must be a valid and signed ECK image.",
							Required:    true,
						},
						"replicas": schema.Int64Attribute{
							Description: "How many replicas in this workload pool.",
							Required:    true,
						},
						"version": schema.StringAttribute{
							Optional: true,
						},
						"autoscaling": schema.SingleNestedAttribute{
							Description: "Configuration options for the autoscaler.",
							Optional:    true,
							Attributes: map[string]schema.Attribute{
								"minimum": schema.Int64Attribute{
									Description: "Minimum number of nodes in this pool.",
									Required:    true,
								},
								"maximum": schema.Int64Attribute{
									Description: "Maximum number of nodes in this pool.",
									Required:    true,
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
