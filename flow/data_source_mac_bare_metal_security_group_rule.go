package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*macBareMetalSecurityGroupRuleDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*macBareMetalSecurityGroupRuleDataSource)(nil)
)

type macBareMetalSecurityGroupRuleDataSourceProtocol struct {
	Number types.Int64  `tfsdk:"number"`
	Name   types.String `tfsdk:"name"`
}

func (c *macBareMetalSecurityGroupRuleDataSourceProtocol) FromNumber(number int) {
	c.Number = types.Int64Value(int64(number))

	if name, found := protocolNumberToName[number]; found {
		c.Name = types.StringValue(name)
	} else {
		c.Name = types.StringNull()
	}
}

type macBareMetalSecurityGroupRuleDataSourcePortRange struct {
	From types.Int64 `tfsdk:"from"`
	To   types.Int64 `tfsdk:"to"`
}

type macBareMetalSecurityGroupRuleDataSourceICMP struct {
	Type types.Int64 `tfsdk:"type"`
	Code types.Int64 `tfsdk:"code"`
}

type macBareMetalSecurityGroupRuleDataSourceData struct {
	ID              types.Int64 `tfsdk:"id"`
	SecurityGroupID types.Int64 `tfsdk:"security_group_id"`

	Direction types.String                                     `tfsdk:"direction"`
	Protocol  *macBareMetalSecurityGroupRuleDataSourceProtocol `tfsdk:"protocol"`

	PortRange *macBareMetalSecurityGroupRuleDataSourcePortRange `tfsdk:"port_range"`
	ICMP      *macBareMetalSecurityGroupRuleDataSourceICMP      `tfsdk:"icmp"`

	IPRange types.String `tfsdk:"ip_range"`
}

func (c *macBareMetalSecurityGroupRuleDataSourceData) FromEntity(securityGroupID int, rule macbaremetal.SecurityGroupRule) {
	c.ID = types.Int64Value(int64(rule.ID))
	c.SecurityGroupID = types.Int64Value(int64(securityGroupID))

	c.Direction = types.StringValue(rule.Direction)
	c.Protocol = &macBareMetalSecurityGroupRuleDataSourceProtocol{}
	c.Protocol.FromNumber(rule.Protocol)

	if rule.FromPort != 0 && rule.ToPort != 0 {
		c.PortRange = &macBareMetalSecurityGroupRuleDataSourcePortRange{
			From: types.Int64Value(int64(rule.FromPort)),
			To:   types.Int64Value(int64(rule.ToPort)),
		}
	}

	if rule.ICMPType != 0 && rule.ICMPCode != 0 {
		c.ICMP = &macBareMetalSecurityGroupRuleDataSourceICMP{
			Type: types.Int64Value(int64(rule.ICMPType)),
			Code: types.Int64Value(int64(rule.ICMPCode)),
		}
	}

	if rule.IPRange == "" {
		c.IPRange = types.StringNull()
	} else {
		c.IPRange = types.StringValue(rule.IPRange)
	}
}

func (c macBareMetalSecurityGroupRuleDataSourceData) AppliesTo(rule macbaremetal.SecurityGroupRule) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(rule.ID) {
		return false
	}

	return true
}

func (c macBareMetalSecurityGroupRuleDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group rule",
				Required:            true,
			},
			"security_group_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group",
				Required:            true,
			},
			"direction": schema.StringAttribute{
				MarkdownDescription: "direction of the security group rule (ingress or egress)",
				Computed:            true,
			},
			"protocol": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"number": schema.Int64Attribute{
						MarkdownDescription: "iana protocol number of the security group rule",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "protocol name of the security group rule",
						Computed:            true,
					},
				},
				MarkdownDescription: "protocol of the security group rule",
				Computed:            true,
			},
			"port_range": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"from": schema.Int64Attribute{
						MarkdownDescription: "starting port of the security group rule",
						Computed:            true,
					},
					"to": schema.Int64Attribute{
						MarkdownDescription: "ending port of the security group rule",
						Computed:            true,
					},
				},
				MarkdownDescription: "port range of the security group rule",
				Computed:            true,
			},
			"icmp": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"type": schema.Int64Attribute{
						MarkdownDescription: "type of the ICMP message",
						Computed:            true,
					},
					"code": schema.Int64Attribute{
						MarkdownDescription: "code of the ICMP message",
						Computed:            true,
					},
				},
				MarkdownDescription: "ICMP message of the security group rule",
				Computed:            true,
			},
			"ip_range": schema.StringAttribute{
				MarkdownDescription: "ip range of the security group rule",
				Computed:            true,
			},
		},
	}
}

func newMacBareMetalSecurityGroupRuleDataSource() datasource.DataSource {
	return &macBareMetalSecurityGroupRuleDataSource{}
}

func (c *macBareMetalSecurityGroupRuleDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mac_bare_metal_security_group_rule"
}

func (c *macBareMetalSecurityGroupRuleDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.securityGroupService = macbaremetal.NewSecurityGroupService(client)
}

type macBareMetalSecurityGroupRuleDataSource struct {
	securityGroupService macbaremetal.SecurityGroupService
}

func (c macBareMetalSecurityGroupRuleDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config macBareMetalSecurityGroupRuleDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(config.SecurityGroupID.ValueInt64())
	ruleID := int(config.ID.ValueInt64())

	list, err := c.securityGroupService.Rules(securityGroupID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list security group rules: %s", err))
		return
	}

	for _, rule := range list.Items {
		if rule.ID == ruleID {
			var state macBareMetalSecurityGroupRuleDataSourceData
			state.FromEntity(securityGroupID, rule)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	response.Diagnostics.AddError("Not Found", fmt.Sprintf("security group rule %d could not be found", ruleID))
}
