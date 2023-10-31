package provider

import (
	"context"
	"os"

	"github.com/eschercloudai/eckctl/pkg/auth"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &eckProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &eckProvider{
			version: version,
		}
	}
}

// eckProvider is the provider implementation.
type eckProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type eckProviderModel struct {
	Host     types.String `tfsdk:"host"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Project  types.String `tfsdk:"project"`
}

// Metadata returns the provider type name.
func (p *eckProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "eck"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *eckProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "URL for the ECK API.  Can also be supplied as the environment variable `ECK_HOST`.",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "Username for the ECK API.  Can also be supplied as the environment variable `ECK_USERNAME`.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Password for the ECK API.  Can also be supplied as the environment variable `ECK_PASSWORD`.",
				Optional:    true,
				Sensitive:   true,
			},
			"project": schema.StringAttribute{
				Description: "OpenStack Project UUID for the ECK API.  Can also be supplied as the environment variable `ECK_PROJECT`.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure prepares an API client for data sources and resources.
func (p *eckProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "🦄 Configuring ECK client")
	// Retrieve provider data from configuration
	var config eckProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "eck_host", config.Host)
	ctx = tflog.SetField(ctx, "eck_username", config.Username)
	ctx = tflog.SetField(ctx, "eck_password", config.Password)
	ctx = tflog.SetField(ctx, "eck_project", config.Project)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "eck_password")

	tflog.Debug(ctx, "Creating ECK client")

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown ECK API Host",
			"The provider cannot create the ECK API client as there is an unknown configuration value for the ECK API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the ECK_HOST environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown ECK API Username",
			"The provider cannot create the ECK API client as there is an unknown configuration value for the ECK API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the ECK_USERNAME environment variable.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown ECK API Password",
			"The provider cannot create the ECK API client as there is an unknown configuration value for the ECK API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the ECK_PASSWORD environment variable.",
		)
	}

	if config.Project.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("project"),
			"Unknown ECK API Project",
			"The provider cannot create the ECK API client as there is an unknown configuration value for the ECK API project. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the ECK_PROJECT environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	host := os.Getenv("ECK_HOST")
	username := os.Getenv("ECK_USERNAME")
	password := os.Getenv("ECK_PASSWORD")
	project := os.Getenv("ECK_PROJECT")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	if !config.Project.IsNull() {
		project = config.Project.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing ECK API Host",
			"The provider cannot create the ECK API client as there is a missing or empty value for the ECK API host. "+
				"Set the host value in the configuration or use the ECK_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing ECK API Username",
			"The provider cannot create the ECK API client as there is a missing or empty value for the ECK API username. "+
				"Set the username value in the configuration or use the ECK_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing ECK API Password",
			"The provider cannot create the ECK API client as there is a missing or empty value for the ECK API password. "+
				"Set the password value in the configuration or use the ECK_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if project == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("project"),
			"Missing ECK API Project",
			"The provider cannot create the ECK API client as there is a missing or empty value for the ECK API project. "+
				"Set the project value in the configuration or use the ECK_PROJECT environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new ECK client using the configuration values
	token, err := auth.GetToken(host, username, password, project, false)
	client, _ := auth.NewClient(host, token, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create ECK API Client",
			"An unexpected error occurred when creating the ECK API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"ECK Client Error: "+err.Error(),
		)
		return
	}

	// Make the ECK client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured ECK client", map[string]any{"success": true})

}

// DataSources defines the data sources implemented in the provider.
func (p *eckProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewControlPlaneDataSource,
		NewClusterDataSource,
		NewKubeconfigDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *eckProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewControlPlaneResource,
		NewClusterResource,
	}
}
