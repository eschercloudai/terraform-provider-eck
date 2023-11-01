package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/eschercloudai/eckctl/pkg/generated"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &clusterDataSource{}
	_ datasource.DataSourceWithConfigure = &clusterDataSource{}
)

// NewClusterDataSource is a helper function to simplify the provider implementation.
func NewClusterDataSource() datasource.DataSource {
	return &clusterDataSource{}
}

// clusterDataSource is the data source implementation.
type clusterDataSource struct {
	client *generated.ClientWithResponses
}

// clusterModel maps clusterModel schema data.
type clusterModel struct {
	ApplicationBundle types.String            `tfsdk:"applicationbundle"`
	ClusterFeatures   *clusterFeaturesModel   `tfsdk:"clusterfeatures"`
	ClusterNetwork    *clusterNetworkModel    `tfsdk:"clusternetwork"`
	ClusterOpenstack  *clusterOpenstackModel  `tfsdk:"clusteropenstack"`
	ControlPlane      *controlPlaneNodesModel `tfsdk:"controlplane"`
	EckCp             types.String            `tfsdk:"eckcp"`
	Kubeconfig        types.String            `tfsdk:"kubeconfig"`
	Name              types.String            `tfsdk:"name"`
	Status            types.String            `tfsdk:"status"`
	WorkloadNodePools []workloadNodePoolModel `tfsdk:"workloadnodepools"`
}

type clusterFeaturesModel struct {
	Autoscaling types.Bool `tfsdk:"autoscaling"`
	Ingress     types.Bool `tfsdk:"ingress"`
	Longhorn    types.Bool `tfsdk:"longhorn"`
	Prometheus  types.Bool `tfsdk:"prometheus"`
	Dashboard   types.Bool `tfsdk:"dashboard"`
}

type controlPlaneNodesModel struct {
	Disk     types.Int64  `tfsdk:"disk"`
	Flavor   types.String `tfsdk:"flavor"`
	Image    types.String `tfsdk:"image"`
	Replicas types.Int64  `tfsdk:"replicas"`
	Version  types.String `tfsdk:"version"`
}

type workloadNodePoolModel struct {
	Name        types.String      `tfsdk:"name"`
	Disk        types.Int64       `tfsdk:"disk"`
	Flavor      types.String      `tfsdk:"flavor"`
	Image       types.String      `tfsdk:"image"`
	Labels      types.Map         `tfsdk:"labels"`
	Replicas    types.Int64       `tfsdk:"replicas"`
	Autoscaling *autoscalingModel `tfsdk:"autoscaling"`
	Version     types.String      `tfsdk:"version"`
}

type autoscalingModel struct {
	MinimumReplicas types.Int64 `tfsdk:"minimum"`
	MaximumReplicas types.Int64 `tfsdk:"maximum"`
}

type clusterNetworkModel struct {
	DnsNameservers types.List   `tfsdk:"dnsnameservers"`
	NodePrefix     types.String `tfsdk:"nodeprefix"`
	PodPrefix      types.String `tfsdk:"podprefix"`
	ServicePrefix  types.String `tfsdk:"serviceprefix"`
}

type clusterOpenstackModel struct {
	ComputeAvailabilityZone types.String `tfsdk:"computeaz"`
	ExternalNetworkID       types.String `tfsdk:"externalnetworkid"`
	SshKeyName              types.String `tfsdk:"sshkey"`
	VolumeAvailabilityZone  types.String `tfsdk:"volumeaz"`
}

// Configure adds the provider configured client to the data source.
func (d *clusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*generated.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T with value of %v. Please report this issue to the provider developers.", req.ProviderData, req.ProviderData),
		)

		return
	}

	d.client = client
}

// Metadata returns the data source type name.
func (d *clusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

// Schema defines the schema for the data source.
func (d *clusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the ECK cluster.",
				Required:    true,
			},
			"applicationbundle": schema.StringAttribute{
				Description: "The version of the bundled components in the cluster.  See https://docs.eschercloud.ai/Kubernetes/Reference/compatibility_matrix for details.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The provisioning status of the cluster.",
			},
			"eckcp": schema.StringAttribute{
				Required:    true,
				Description: "The associated ECK Control Plane for the cluster.",
			},
			"kubeconfig": schema.StringAttribute{
				Computed:    true,
				Description: "The kubeconfig for the cluster.",
			},
			"controlplane": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"disk": schema.Int64Attribute{
						Computed:    true,
						Description: "Whether to use a dedicated persistent volume for control plane nodes. It is recommended to leave this unchecked, as ephemeral storage provides higher performance for Kubernetes' etcd database. If left unset, the default ephemeral storage size of 20GB is used.",
					},
					"flavor": schema.StringAttribute{
						Computed:    true,
						Description: "The flavor (size) of the machine.",
					},
					"image": schema.StringAttribute{
						Computed:    true,
						Description: "Which OS image to use.  Must be a verified and signed ECK image",
					},
					"replicas": schema.Int64Attribute{
						Computed:    true,
						Description: "How many replicas to provision in a control plane.  Must be an odd number, 3 is recommended.",
					},
					"version": schema.StringAttribute{
						Computed:    true,
						Description: "The version of Kubernetes.  Must match the version bundled with the OS image.",
					},
				},
			},
			"clusternetwork": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"dnsnameservers": schema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
						Description: "A list of DNS nameservers used by the OS.",
					},
					"nodeprefix": schema.StringAttribute{
						Computed:    true,
						Description: "The CIDR-formatted IP address range to be used by Nodes in the cluster.",
					},
					"podprefix": schema.StringAttribute{
						Computed:    true,
						Description: "The CIDR-formatted IP address range to be used by Pods in the cluster.",
					},
					"serviceprefix": schema.StringAttribute{
						Computed:    true,
						Description: "The CIDR-formatted IP address range to be used by Services in the cluster.",
					},
				},
			},
			"clusteropenstack": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Features which dictate OpenStack-specific behaviour and options.",
				Attributes: map[string]schema.Attribute{
					"computeaz": schema.StringAttribute{
						Computed:    true,
						Description: "OpenStack Compute Availability Zone. Defaults to `nova`.",
					},
					"externalnetworkid": schema.StringAttribute{
						Computed:    true,
						Description: "UUID of the external network.",
					},
					"sshkey": schema.StringAttribute{
						Computed:    true,
						Description: "SSH key associated with the instance.",
					},
					"volumeaz": schema.StringAttribute{
						Computed:    true,
						Description: "OpenStack Cinder Availability Zone. Defaults to `nova`.",
					},
				},
			},
			"clusterfeatures": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"autoscaling": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Enables Cluster Autoscaler, required for autoscaling workload pools.",
					},
					"ingress": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Whether to deploy the NGINX Ingress Controller.",
					},
					"longhorn": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Whether to enable Longhorn for persistent storage, which includes support for RWX.",
					},
					"prometheus": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Whether to enable the Prometheus Operator for monitoring.",
					},
					"dashboard": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Whether to enable the Kubernetes Dashboard.",
					},
				},
			},
			"workloadnodepools": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the workload pool.",
						},
						"disk": schema.Int64Attribute{
							Computed:    true,
							Description: "Size of disk for the node.  Defaults to 50GiB.",
						},
						"flavor": schema.StringAttribute{
							Computed:    true,
							Description: "OpenStack flavor (size) for nodes in this pool.",
						},
						"image": schema.StringAttribute{
							Computed:    true,
							Description: "Operating system image to use.  Must be a valid and signed ECK image.",
						},
						"labels": schema.MapAttribute{
							ElementType: types.StringType,
							Computed:    true,
							Optional:    true,
							Description: "A map of Kubernetes labels to be applied to each node in the pool.",
						},
						"replicas": schema.Int64Attribute{
							Computed:    true,
							Description: "How many replicas in this workload pool.",
						},
						"version": schema.StringAttribute{
							Computed:    true,
							Description: "The version of Kubernetes.  Must match the version bundled with the OS image.",
						},
						"autoscaling": schema.SingleNestedAttribute{
							Computed:    true,
							Description: "Configuration options for the autoscaler.",
							Attributes: map[string]schema.Attribute{
								"minimum": schema.Int64Attribute{
									Computed:    true,
									Description: "Minimum number of nodes in this pool.",
								},
								"maximum": schema.Int64Attribute{
									Computed:    true,
									Description: "Maximum number of nodes in this pool.",
								},
							},
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *clusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state clusterModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	r, err := d.client.GetApiV1ControlplanesControlPlaneNameClustersClusterName(ctx, state.EckCp.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to retrieve cluster information",
			err.Error(),
		)
		return
	}

	if r.StatusCode != http.StatusOK {
		fmt.Printf("Error retrieving cluster information, %v", r.StatusCode)
		return
	}

	cluster := generated.KubernetesCluster{}
	err = json.NewDecoder(r.Body).Decode(&cluster)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read cluster information",
			"An error occurred while parsing the response from the ECK API."+
				"JSON Error: "+err.Error(),
		)
	}

	var kubeconfig string
	if cluster.Status.Status == "Provisioned" {
		kubeconfig = getKubeconfig(*d.client, ctx, state.EckCp.ValueString(), cluster.Name)
	} else {
		kubeconfig = ""
	}

	// Map response body to model
	state = generateClusterModel(ctx, cluster, state.EckCp.ValueString(), string(kubeconfig))

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
