package provider

import (
	"context"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &ContentFlowProvider{}

// ContentFlowProvider talks to a ContentFlow dashboard's token-authenticated
// /api/v1 JSON API (see services/dashboard/app/api_routes.py in the
// ContentFlow repo) to manage assets.
type ContentFlowProvider struct {
	version string
}

type contentFlowProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	APIToken types.String `tfsdk:"api_token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ContentFlowProvider{version: version}
	}
}

func (p *ContentFlowProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "contentflow"
	resp.Version = p.version
}

func (p *ContentFlowProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages assets hosted by a self-hosted ContentFlow dashboard through its /api/v1 JSON API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional: true,
				Description: "Base URL of the ContentFlow dashboard service, e.g. " +
					"\"https://dashboard.example.com\". Falls back to the " +
					"CONTENTFLOW_ENDPOINT environment variable.",
			},
			"api_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Bearer token for the dashboard's /api/v1 API -- must match " +
					"DASHBOARD_API_TOKEN on the server. Falls back to the " +
					"CONTENTFLOW_API_TOKEN environment variable.",
			},
		},
	}
}

func (p *ContentFlowProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data contentFlowProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := data.Endpoint.ValueString()
	if endpoint == "" {
		endpoint = os.Getenv("CONTENTFLOW_ENDPOINT")
	}
	if endpoint == "" {
		resp.Diagnostics.AddError(
			"Missing ContentFlow endpoint",
			"Set the provider's `endpoint` attribute or the CONTENTFLOW_ENDPOINT "+
				"environment variable to the dashboard's base URL.",
		)
	}

	apiToken := data.APIToken.ValueString()
	if apiToken == "" {
		apiToken = os.Getenv("CONTENTFLOW_API_TOKEN")
	}
	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing ContentFlow API token",
			"Set the provider's `api_token` attribute or the CONTENTFLOW_API_TOKEN "+
				"environment variable. This must match DASHBOARD_API_TOKEN on the server.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client := NewClient(endpoint, apiToken, http.DefaultClient)
	resp.ResourceData = client
}

func (p *ContentFlowProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAssetResource,
	}
}

func (p *ContentFlowProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
