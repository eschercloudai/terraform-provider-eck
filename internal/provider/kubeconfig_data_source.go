package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/eschercloudai/eckctl/pkg/generated"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &kubeconfigDataSource{}
	_ datasource.DataSourceWithConfigure = &kubeconfigDataSource{}
)

// NewKubeconfigDataSource is a helper function to simplify the provider implementation.
func NewKubeconfigDataSource() datasource.DataSource {
	return &kubeconfigDataSource{}
}

// coffeesDataSource is the data source implementation.
type kubeconfigDataSource struct {
	client *generated.ClientWithResponses
}

// Metadata returns the data source type name.
func (d *kubeconfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubeconfig"
}

// Schema defines the schema for the data source.
func (d *kubeconfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"kubeconfig": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

type kubeconfigModel struct {
	Kubeconfig types.String `tfsdk:"kubeconfig"`
}

// Configure adds the provider configured client to the data source.
func (d *kubeconfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read refreshes the Terraform state with the latest data.
func (d *kubeconfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	r, err := d.client.GetApiV1ControlplanesControlPlaneNameClustersClusterNameKubeconfig(ctx, "tftest", "terratest")
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}

	state := kubeconfigModel{
		Kubeconfig: types.StringValue((string(body))),
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}
