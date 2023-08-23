package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	Flavor      types.String      `tfsdk:"flavor"`
	Image       types.String      `tfsdk:"image"`
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
				Required: true,
			},
			"applicationbundle": schema.StringAttribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"eckcp": schema.StringAttribute{
				Required: true,
			},
			"kubeconfig": schema.StringAttribute{
				Computed: true,
			},
			"controlplane": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"disk": schema.Int64Attribute{
						Computed: true,
					},
					"flavor": schema.StringAttribute{
						Computed: true,
					},
					"image": schema.StringAttribute{
						Computed: true,
					},
					"replicas": schema.Int64Attribute{
						Computed: true,
					},
					"version": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"clusternetwork": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"dnsnameservers": schema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
					},
					"nodeprefix": schema.StringAttribute{
						Computed: true,
					},
					"podprefix": schema.StringAttribute{
						Computed: true,
					},
					"serviceprefix": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"clusteropenstack": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"computeaz": schema.StringAttribute{
						Computed: true,
					},
					"externalnetworkid": schema.StringAttribute{
						Computed: true,
					},
					"sshkey": schema.StringAttribute{
						Computed: true,
					},
					"volumeaz": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"clusterfeatures": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"autoscaling": schema.BoolAttribute{
						Computed: true,
					},
				},
			},
			"workloadnodepools": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed: true,
						},
						"flavor": schema.StringAttribute{
							Computed: true,
						},
						"image": schema.StringAttribute{
							Computed: true,
						},
						"replicas": schema.Int64Attribute{
							Computed: true,
						},
						"version": schema.StringAttribute{
							Computed: true,
						},
						"autoscaling": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"minimum": schema.Int64Attribute{
									Computed: true,
								},
								"maximum": schema.Int64Attribute{
									Computed: true,
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}

	cluster := generated.KubernetesCluster{}
	err = json.Unmarshal(body, &cluster)
	if err != nil {
		return
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
