package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ datasource.DataSource              = (*macBareMetalElasticIPDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*macBareMetalElasticIPDataSource)(nil)
)

type macBareMetalElasticIPDataSourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	LocationID types.Int64  `tfsdk:"location_id"`
	PublicIP   types.String `tfsdk:"public_ip"`
}

func (c *macBareMetalElasticIPDataSourceData) FromEntity(elasticIP macbaremetal.ElasticIP) {
	c.ID = types.Int64Value(int64(elasticIP.ID))
	c.LocationID = types.Int64Value(int64(elasticIP.Location.ID))
	c.PublicIP = types.StringValue(elasticIP.PublicIP)
}

func (c macBareMetalElasticIPDataSourceData) AppliesTo(elasticIP macbaremetal.ElasticIP) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(elasticIP.ID) {
		return false
	}

	if !c.LocationID.IsNull() && c.LocationID.ValueInt64() != int64(elasticIP.Location.ID) {
		return false
	}

	if !c.PublicIP.IsNull() && c.PublicIP.ValueString() != elasticIP.PublicIP {
		return false
	}

	return true
}

func (c macBareMetalElasticIPDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the elastic ip",
				Optional:            true,
				Computed:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "location of the elastic ip",
				Optional:            true,
				Computed:            true,
			},
			"public_ip": schema.StringAttribute{
				MarkdownDescription: "public ip address",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newMacBareMetalElasticIPDataSource() datasource.DataSource {
	return &macBareMetalElasticIPDataSource{}
}

func (c *macBareMetalElasticIPDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mac_bare_metal_elastic_ip"
}

func (c *macBareMetalElasticIPDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.elasticIPService = macbaremetal.NewElasticIPService(client)
}

type macBareMetalElasticIPDataSource struct {
	elasticIPService macbaremetal.ElasticIPService
}

func (c macBareMetalElasticIPDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config macBareMetalElasticIPDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.elasticIPService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list elastic ips: %s", err))
		return
	}

	elasticIP, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find elastic ip: %s", err))
		return
	}

	var state macBareMetalElasticIPDataSourceData
	state.FromEntity(elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
