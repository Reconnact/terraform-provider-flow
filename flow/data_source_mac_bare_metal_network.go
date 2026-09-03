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
	_ datasource.DataSource              = (*macBareMetalNetworkDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*macBareMetalNetworkDataSource)(nil)
)

type macBareMetalNetworkDataSourceAllocationPool struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type macBareMetalNetworkDataSourceData struct {
	ID                types.Int64                                  `tfsdk:"id"`
	Name              types.String                                 `tfsdk:"name"`
	CIDR              types.String                                 `tfsdk:"cidr"`
	LocationID        types.Int64                                  `tfsdk:"location_id"`
	DomainNameServers []types.String                               `tfsdk:"domain_name_servers"`
	AllocationPool    *macBareMetalNetworkDataSourceAllocationPool `tfsdk:"allocation_pool"`
	GatewayIP         types.String                                 `tfsdk:"gateway_ip"`
}

func (c *macBareMetalNetworkDataSourceData) FromEntity(network macbaremetal.Network) {
	c.ID = types.Int64Value(int64(network.ID))
	c.Name = types.StringValue(network.Name)
	c.CIDR = types.StringValue(network.Subnet)
	c.LocationID = types.Int64Value(int64(network.Location.ID))
	c.GatewayIP = types.StringValue(network.GatewayIP)

	c.AllocationPool = &macBareMetalNetworkDataSourceAllocationPool{
		Start: types.StringValue(network.AllocationPoolStart),
		End:   types.StringValue(network.AllocationPoolEnd),
	}

	c.DomainNameServers = make([]types.String, len(network.DomainNameServers))
	for idx, domainNameServer := range network.DomainNameServers {
		c.DomainNameServers[idx] = types.StringValue(domainNameServer)
	}
}

func (c macBareMetalNetworkDataSourceData) AppliesTo(network macbaremetal.Network) bool {
	if !c.ID.IsNull() && network.ID != int(c.ID.ValueInt64()) {
		return false
	}

	if !c.Name.IsNull() && network.Name != c.Name.ValueString() {
		return false
	}

	return true
}

func (c macBareMetalNetworkDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the network",
				Optional:            true,
				Computed:            true,
			},
			"cidr": schema.StringAttribute{
				MarkdownDescription: "CIDR of the network",
				Computed:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Optional:            true,
				Computed:            true,
			},
			"domain_name_servers": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "list of domain name servers",
				Computed:            true,
			},
			"allocation_pool": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"start": schema.StringAttribute{
						MarkdownDescription: "start of the allocation pool",
						Computed:            true,
					},
					"end": schema.StringAttribute{
						MarkdownDescription: "end of the allocation pool",
						Computed:            true,
					},
				},
				MarkdownDescription: "allocation pool",
				Computed:            true,
			},
			"gateway_ip": schema.StringAttribute{
				MarkdownDescription: "gateway IP of the network",
				Computed:            true,
			},
		},
	}
}

func newMacBareMetalNetworkDataSource() datasource.DataSource {
	return &macBareMetalNetworkDataSource{}
}

func (c *macBareMetalNetworkDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mac_bare_metal_network"
}

func (c *macBareMetalNetworkDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.networkService = macbaremetal.NewNetworkService(client)
}

type macBareMetalNetworkDataSource struct {
	networkService macbaremetal.NetworkService
}

func (c macBareMetalNetworkDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config macBareMetalNetworkDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.networkService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list networks: %s", err))
		return
	}

	network, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find network: %s", err))
		return
	}

	var state macBareMetalNetworkDataSourceData
	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
