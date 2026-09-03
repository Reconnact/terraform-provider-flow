package flow

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/flowswiss/goclient"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = (*flowProvider)(nil)

type Option func(p *flowProvider)

func WithVersion(version string) Option {
	return func(p *flowProvider) {
		p.version = version
	}
}

func WithDefaultEndpoint(endpoint string) Option {
	return func(p *flowProvider) {
		p.defaultEndpoint = endpoint
	}
}

func New(opts ...Option) provider.Provider {
	p := &flowProvider{
		version:         "dev",
		defaultEndpoint: "https://api.flow.swiss/",
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

type flowProvider struct {
	version         string
	defaultEndpoint string
}

type providerData struct {
	Token        types.String `tfsdk:"token"`
	Endpoint     types.String `tfsdk:"endpoint"`
	RetryTimeout types.String `tfsdk:"retry_timeout"`
}

func (p *flowProvider) Metadata(ctx context.Context, request provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "flow"
	response.Version = p.version
}

func (p *flowProvider) Schema(ctx context.Context, request provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				MarkdownDescription: "authentication token for the flow api",
				Optional:            true,
				Sensitive:           true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "endpoint for the flow api",
				Optional:            true,
			},
			"retry_timeout": schema.StringAttribute{
				MarkdownDescription: "how long a failing api call is retried before the error is reported, as a duration such as `90s` or `2m` (default `90s`, `0` disables retries). can also be set with the `FLOW_RETRY_TIMEOUT` environment variable",
				Optional:            true,
			},
		},
	}
}

func (p *flowProvider) Configure(ctx context.Context, request provider.ConfigureRequest, response *provider.ConfigureResponse) {
	var data providerData
	diagnostics := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	if data.Token.IsNull() {
		if val, ok := os.LookupEnv("FLOW_TOKEN"); ok {
			data.Token = types.StringValue(val)
		} else {
			response.Diagnostics.AddError(
				"Missing Token",
				"The token is missing. Please set the token in the provider configuration or set the FLOW_TOKEN environment variable.",
			)
			return
		}
	}

	if data.Endpoint.IsNull() {
		data.Endpoint = types.StringValue(p.defaultEndpoint)

		if val, ok := os.LookupEnv("FLOW_ENDPOINT"); ok {
			data.Endpoint = types.StringValue(val)
		}
	}

	if data.RetryTimeout.IsNull() {
		if val, ok := os.LookupEnv("FLOW_RETRY_TIMEOUT"); ok {
			data.RetryTimeout = types.StringValue(val)
		}
	}

	if !data.RetryTimeout.IsNull() {
		timeout, err := parseRetryTimeout(data.RetryTimeout.ValueString())
		if err != nil {
			response.Diagnostics.AddAttributeError(
				path.Root("retry_timeout"),
				"Invalid Retry Timeout",
				err.Error(),
			)
			return
		}

		defaultRetryPolicy.Timeout = timeout
	}

	client := goclient.NewClient(
		goclient.WithToken(data.Token.ValueString()),
		goclient.WithBase(data.Endpoint.ValueString()),
		goclient.WithUserAgent(fmt.Sprintf("terraform-provider-flow/%s", p.version)),

		goclient.WithHTTPClientOption(installTransport),
	)

	response.ResourceData = client
	response.DataSourceData = client
}

func (p *flowProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newComputeCertificateResource,
		newComputeElasticIPResource,
		newComputeElasticIPServerAttachmentResource,
		newComputeKeyPairResource,
		newComputeLoadBalancerResource,
		newComputeLoadBalancerMemberResource,
		newComputeLoadBalancerPoolResource,
		newComputeNetworkResource,
		newComputeNetworkInterfaceResource,
		newComputeRouterResource,
		newComputeRouterInterfaceResource,
		newComputeRouterRouteResource,
		newComputeSecurityGroupResource,
		newComputeSecurityGroupRuleResource,
		newComputeServerResource,
		newComputeSnapshotResource,
		newComputeVolumeResource,
		newComputeVolumeAttachmentResource,

		newKubernetesClusterResource,

		newMacBareMetalDeviceResource,
		newMacBareMetalElasticIPResource,
		newMacBareMetalElasticIPDeviceAttachmentResource,
		newMacBareMetalNetworkResource,
		newMacBareMetalSecurityGroupResource,
		newMacBareMetalSecurityGroupRuleResource,
	}
}

func (p *flowProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newLocationDataSource,
		newModuleDataSource,
		newProductDataSource,

		newComputeCertificateDataSource,
		newComputeElasticIPDataSource,
		newComputeImageDataSource,
		newComputeKeyPairDataSource,
		newComputeLoadBalancerAlgorithmDataSource,
		newComputeLoadBalancerHealthCheckTypeDataSource,
		newComputeLoadBalancerMemberDataSource,
		newComputeLoadBalancerPoolDataSource,
		newComputeLoadBalancerProtocolDataSource,
		newComputeNetworkDataSource,
		newComputeNetworkInterfaceDataSource,
		newComputeRouterDataSource,
		newComputeRouterInterfaceDataSource,
		newComputeRouterRouteDataSource,
		newComputeSecurityGroupDataSource,
		newComputeSecurityGroupRuleDataSource,
		newComputeServerDataSource,
		newComputeSnapshotDataSource,
		newComputeVolumeDataSource,

		newKubernetesClusterDataSource,
		newKubernetesKubeConfigDataSource,

		newMacBareMetalElasticIPDataSource,
		newMacBareMetalNetworkDataSource,
		newMacBareMetalSecurityGroupDataSource,
		newMacBareMetalSecurityGroupRuleDataSource,
	}
}

// the framework configures resources and data sources before the provider is
// configured too (validate, plan without a token) — ProviderData is nil then
// and the resource must stay unconfigured without complaint
func clientFromProviderData(data any, diagnostics *diag.Diagnostics) (goclient.Client, bool) {
	if data == nil {
		return goclient.Client{}, false
	}

	client, ok := data.(goclient.Client)
	if !ok {
		diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("While configuring the data source or resource, an unexpected provider data type (%T) was received. This is always a bug in the provider code and should be reported to the provider developers.", data),
		)
		return goclient.Client{}, false
	}

	return client, true
}

type logTransport struct {
	base http.RoundTripper
}

func (l logTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	additionalContext := map[string]interface{}{
		"method": req.Method,
		"url":    req.URL.String(),
	}

	res, err := l.transport().RoundTrip(req)

	if err == nil {
		additionalContext["request_id"] = res.Header.Get("X-Request-ID")

		msg := fmt.Sprintf("request to `%s %s` resulted in `%s`", req.Method, req.URL.String(), res.Status)
		tflog.Trace(req.Context(), msg, additionalContext)
	} else {
		msg := fmt.Sprintf("request to `%s %s` resulted in `%s`", req.Method, req.URL.String(), err)
		tflog.Trace(req.Context(), msg, additionalContext)
	}

	return res, err
}

func (l logTransport) transport() http.RoundTripper {
	if l.base == nil {
		return http.DefaultTransport
	}

	return l.base
}

func installTransport(c *http.Client) {
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.ResponseHeaderTimeout = responseHeaderTimeout

	// goclient's WithToken put its auth transport in front of the default one —
	// replacing it drops the authorization header
	var inner http.RoundTripper = base
	switch t := c.Transport.(type) {
	case goclient.AuthTransport:
		t.Base = base
		inner = t
	case nil:
	default:
		inner = t
	}

	// the read retry sits outside the log transport so every attempt is traced
	c.Transport = readRetryTransport{base: logTransport{base: inner}}
}
