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
	_ datasource.DataSource              = &controlPlaneDataSource{}
	_ datasource.DataSourceWithConfigure = &controlPlaneDataSource{}
)

// NewControlPlaneDataSource is a helper function to simplify the provider implementation.
func NewControlPlaneDataSource() datasource.DataSource {
	return &controlPlaneDataSource{}
}

// controlPlaneDataSource is the data source implementation.
type controlPlaneDataSource struct {
	client *generated.ClientWithResponses
}

// Configure adds the provider configured client to the data source.
func (d *controlPlaneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *controlPlaneDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_controlplanes"
}

// Schema defines the schema for the data source.
func (d *controlPlaneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"controlplanes": schema.ListNestedAttribute{
				Computed:    true,
				Description: "A list of ECK Control Planes.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the ECK Control Plane.",
						},
						"applicationbundle": schema.SingleNestedAttribute{
							Required: true,
							Attributes: map[string]schema.Attribute{
								"version": schema.StringAttribute{
									Computed:    true,
									Description: "The version of the ECK Control Plane.",
								},
								"autoupgrade": schema.BoolAttribute{
									Required:    true,
									Description: "Whether automatic upgrades of the ECK Control Plane are enabled.",
								},
							},
						},
					},
				},
			},
		},
	}
}

// controlPlaneDataSourceModel maps the data source schema data.
type controlPlaneDataSourceModel struct {
	Controlplanes []controlPlaneModel `tfsdk:"controlplanes"`
}

// controlPlaneModel maps controlPlane schema data.
type controlPlaneModel struct {
	Name              types.String           `tfsdk:"name"`
	ApplicationBundle applicationBundleModel `tfsdk:"applicationbundle"`
}

type applicationBundleModel struct {
	Version     types.String `tfsdk:"version"`
	AutoUpgrade types.Bool   `tfsdk:"autoupgrade"`
}

func IsDaysOfWeekSet(aba *generated.ApplicationBundleAutoUpgrade) bool {
	if aba == nil {
		return false
	}
	return aba.DaysOfWeek != nil
}

// Read refreshes the Terraform state with the latest data.
func (d *controlPlaneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state controlPlaneDataSourceModel

	r, err := d.client.GetApiV1Controlplanes(ctx)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		fmt.Printf("Error retrieving control plane information, %v", r.StatusCode)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}

	controlPlanes := generated.ControlPlanes{}

	err = json.Unmarshal(body, &controlPlanes)
	if err != nil {
		fmt.Println(err)
	}

	// Map response body to model
	for _, controlPlane := range controlPlanes {
		controlPlaneState := controlPlaneModel{
			Name: types.StringValue(controlPlane.Name),
			ApplicationBundle: applicationBundleModel{
				Version:     types.StringValue(controlPlane.ApplicationBundle.Name),
				AutoUpgrade: types.BoolValue(IsDaysOfWeekSet(controlPlane.ApplicationBundleAutoUpgrade)),
			},
		}

		state.Controlplanes = append(state.Controlplanes, controlPlaneState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
