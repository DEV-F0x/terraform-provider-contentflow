package provider

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"strconv"

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
	Insecure types.Bool   `tfsdk:"insecure"`
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
			"insecure": schema.BoolAttribute{
				Optional: true,
				Description: "Skip TLS certificate verification when connecting to " +
					"the dashboard -- for self-signed or internal-CA certificates. " +
					"Defaults to false. Falls back to the CONTENTFLOW_INSECURE " +
					"environment variable (any value accepted by Go's " +
					"strconv.ParseBool, e.g. \"true\"/\"1\").",
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

	insecure := data.Insecure.ValueBool()
	if data.Insecure.IsNull() {
		if v := os.Getenv("CONTENTFLOW_INSECURE"); v != "" {
			parsed, err := strconv.ParseBool(v)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid CONTENTFLOW_INSECURE value",
					"Expected a boolean (e.g. \"true\"/\"false\"), got: "+v,
				)
				return
			}
			insecure = parsed
		}
	}

	httpClient := http.DefaultClient
	if insecure {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 -- opt-in for self-signed/internal-CA dashboards
			},
		}
	}

	client := NewClient(endpoint, apiToken, httpClient)
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
