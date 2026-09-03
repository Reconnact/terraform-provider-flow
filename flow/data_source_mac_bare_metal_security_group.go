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
	_ datasource.DataSource              = (*macBareMetalSecurityGroupDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*macBareMetalSecurityGroupDataSource)(nil)
)

type macBareMetalSecurityGroupDataSourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	NetworkID types.Int64  `tfsdk:"network_id"`
}

func (c *macBareMetalSecurityGroupDataSourceData) FromEntity(securityGroup macbaremetal.SecurityGroup) {
	c.ID = types.Int64Value(int64(securityGroup.ID))
	c.Name = types.StringValue(securityGroup.Name)
	c.NetworkID = types.Int64Value(int64(securityGroup.Network.ID))
}

func (c macBareMetalSecurityGroupDataSourceData) AppliesTo(securityGroup macbaremetal.SecurityGroup) bool {
	if !c.ID.IsNull() && securityGroup.ID != int(c.ID.ValueInt64()) {
		return false
	}

	if !c.Name.IsNull() && securityGroup.Name != c.Name.ValueString() {
		return false
	}

	if !c.NetworkID.IsNull() && securityGroup.Network.ID != int(c.NetworkID.ValueInt64()) {
		return false
	}

	return true
}

func (c macBareMetalSecurityGroupDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the security group",
				Optional:            true,
				Computed:            true,
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newMacBareMetalSecurityGroupDataSource() datasource.DataSource {
	return &macBareMetalSecurityGroupDataSource{}
}

func (c *macBareMetalSecurityGroupDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mac_bare_metal_security_group"
}

func (c *macBareMetalSecurityGroupDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.securityGroupService = macbaremetal.NewSecurityGroupService(client)
}

type macBareMetalSecurityGroupDataSource struct {
	securityGroupService macbaremetal.SecurityGroupService
}

func (c macBareMetalSecurityGroupDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config macBareMetalSecurityGroupDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.securityGroupService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list security groups: %s", err))
		return
	}

	securityGroup, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find security group: %s", err))
		return
	}

	var state macBareMetalSecurityGroupDataSourceData
	state.FromEntity(securityGroup)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
