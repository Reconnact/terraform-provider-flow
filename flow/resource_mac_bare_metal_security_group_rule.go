package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/validators"
)

var (
	_ resource.Resource                     = (*macBareMetalSecurityGroupRuleResource)(nil)
	_ resource.ResourceWithConfigure        = (*macBareMetalSecurityGroupRuleResource)(nil)
	_ resource.ResourceWithConfigValidators = (*macBareMetalSecurityGroupRuleResource)(nil)
)

var macBareMetalProtocolNumberToName = map[int]string{
	macbaremetal.ProtocolAny:  "any",
	macbaremetal.ProtocolICMP: "icmp",
	macbaremetal.ProtocolUDP:  "udp",
	macbaremetal.ProtocolTCP:  "tcp",
}

var macBareMetalProtocolNamesToNumber = map[string]int{
	"any":  macbaremetal.ProtocolAny,
	"icmp": macbaremetal.ProtocolICMP,
	"udp":  macbaremetal.ProtocolUDP,
	"tcp":  macbaremetal.ProtocolTCP,
}

type macBareMetalSecurityGroupRuleResourceProtocol struct {
	Number types.Int64  `tfsdk:"number"`
	Name   types.String `tfsdk:"name"`
}

func (r *macBareMetalSecurityGroupRuleResourceProtocol) FromNumber(number int) {
	r.Number = types.Int64Value(int64(number))

	if name, found := macBareMetalProtocolNumberToName[number]; found {
		r.Name = types.StringValue(name)
	} else {
		r.Name = types.StringNull()
	}
}

func (r macBareMetalSecurityGroupRuleResourceProtocol) ToNumber() int {
	if !r.Number.IsNull() {
		return int(r.Number.ValueInt64())
	}

	if !r.Name.IsNull() {
		return macBareMetalProtocolNamesToNumber[r.Name.ValueString()]
	}

	return 0
}

type macBareMetalSecurityGroupRuleResourcePortRange struct {
	From types.Int64 `tfsdk:"from"`
	To   types.Int64 `tfsdk:"to"`
}

type macBareMetalSecurityGroupRuleResourceICMP struct {
	Type types.Int64 `tfsdk:"type"`
	Code types.Int64 `tfsdk:"code"`
}

type macBareMetalSecurityGroupRuleResourceData struct {
	ID              types.Int64 `tfsdk:"id"`
	SecurityGroupID types.Int64 `tfsdk:"security_group_id"`

	Direction types.String                                   `tfsdk:"direction"`
	Protocol  *macBareMetalSecurityGroupRuleResourceProtocol `tfsdk:"protocol"`

	PortRange *macBareMetalSecurityGroupRuleResourcePortRange `tfsdk:"port_range"`
	ICMP      *macBareMetalSecurityGroupRuleResourceICMP      `tfsdk:"icmp"`

	IPRange types.String `tfsdk:"ip_range"`
}

func (r *macBareMetalSecurityGroupRuleResourceData) FromEntity(securityGroupID int, rule macbaremetal.SecurityGroupRule) {
	r.ID = types.Int64Value(int64(rule.ID))
	r.SecurityGroupID = types.Int64Value(int64(securityGroupID))

	r.Direction = types.StringValue(rule.Direction)
	r.Protocol = &macBareMetalSecurityGroupRuleResourceProtocol{}
	r.Protocol.FromNumber(rule.Protocol)

	if rule.Protocol == macbaremetal.ProtocolTCP || rule.Protocol == macbaremetal.ProtocolUDP {
		r.PortRange = &macBareMetalSecurityGroupRuleResourcePortRange{
			From: types.Int64Value(int64(rule.FromPort)),
			To:   types.Int64Value(int64(rule.ToPort)),
		}
	}

	if rule.Protocol == macbaremetal.ProtocolICMP {
		r.ICMP = &macBareMetalSecurityGroupRuleResourceICMP{
			Type: types.Int64Value(int64(rule.ICMPType)),
			Code: types.Int64Value(int64(rule.ICMPCode)),
		}
	}

	if rule.IPRange == "" {
		r.IPRange = types.StringNull()
	} else {
		r.IPRange = types.StringValue(rule.IPRange)
	}
}

func (r macBareMetalSecurityGroupRuleResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group rule",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"security_group_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"direction": schema.StringAttribute{
				MarkdownDescription: "direction of the security group rule (ingress or egress)",
				Required:            true,
			},
			"protocol": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"number": schema.Int64Attribute{
						MarkdownDescription: "iana protocol number of the security group rule",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "protocol name of the security group rule",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
				MarkdownDescription: "protocol of the security group rule",
				Required:            true,
			},
			"port_range": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"from": schema.Int64Attribute{
						MarkdownDescription: "starting port of the security group rule",
						Required:            true,
					},
					"to": schema.Int64Attribute{
						MarkdownDescription: "ending port of the security group rule",
						Required:            true,
					},
				},
				MarkdownDescription: "port range filter of the security group rule",
				Optional:            true,
			},
			"icmp": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"type": schema.Int64Attribute{
						MarkdownDescription: "type of the ICMP message",
						Required:            true,
					},
					"code": schema.Int64Attribute{
						MarkdownDescription: "code of the ICMP message",
						Required:            true,
					},
				},
				MarkdownDescription: "ICMP message filter of the security group rule",
				Optional:            true,
			},
			"ip_range": schema.StringAttribute{
				MarkdownDescription: "ip range of the security group rule",
				Optional:            true,
			},
		},
	}
}

func newMacBareMetalSecurityGroupRuleResource() resource.Resource {
	return &macBareMetalSecurityGroupRuleResource{}
}

func (r *macBareMetalSecurityGroupRuleResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mac_bare_metal_security_group_rule"
}

func (r *macBareMetalSecurityGroupRuleResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	r.securityGroupService = macbaremetal.NewSecurityGroupService(client)
}

type macBareMetalSecurityGroupRuleResource struct {
	securityGroupService macbaremetal.SecurityGroupService
}

func (r macBareMetalSecurityGroupRuleResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config macBareMetalSecurityGroupRuleResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(config.SecurityGroupID.ValueInt64())
	create := macbaremetal.SecurityGroupRuleOptions{
		Direction: config.Direction.ValueString(),
		Protocol:  config.Protocol.ToNumber(),
		IPRange:   config.IPRange.ValueString(),
	}

	if config.PortRange != nil {
		create.FromPort = int(config.PortRange.From.ValueInt64())
		create.ToPort = int(config.PortRange.To.ValueInt64())
	}

	if config.ICMP != nil {
		create.ICMPType = int(config.ICMP.Type.ValueInt64())
		create.ICMPCode = int(config.ICMP.Code.ValueInt64())
	}

	rule, err := r.securityGroupService.Rules(securityGroupID).Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create security group rule: %s", err))
		return
	}

	var state macBareMetalSecurityGroupRuleResourceData
	state.FromEntity(securityGroupID, rule)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalSecurityGroupRuleResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state macBareMetalSecurityGroupRuleResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(state.SecurityGroupID.ValueInt64())
	ruleID := int(state.ID.ValueInt64())

	list, err := r.securityGroupService.Rules(securityGroupID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list security group rules: %s", err))
		return
	}

	for _, rule := range list.Items {
		if rule.ID == ruleID {
			state.FromEntity(securityGroupID, rule)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	response.Diagnostics.AddError("Not Found", fmt.Sprintf("security group rule %d could not be found", ruleID))
}

func (r macBareMetalSecurityGroupRuleResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state macBareMetalSecurityGroupRuleResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config macBareMetalSecurityGroupRuleResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(config.SecurityGroupID.ValueInt64())
	ruleID := int(state.ID.ValueInt64())

	update := macbaremetal.SecurityGroupRuleOptions{
		Direction: config.Direction.ValueString(),
		Protocol:  config.Protocol.ToNumber(),
		IPRange:   config.IPRange.ValueString(),
	}

	if config.PortRange != nil {
		update.FromPort = int(config.PortRange.From.ValueInt64())
		update.ToPort = int(config.PortRange.To.ValueInt64())
	}

	if config.ICMP != nil {
		update.ICMPType = int(config.ICMP.Type.ValueInt64())
		update.ICMPCode = int(config.ICMP.Code.ValueInt64())
	}

	rule, err := r.securityGroupService.Rules(securityGroupID).Update(ctx, ruleID, update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security group rule: %s", err))
		return
	}

	state.FromEntity(securityGroupID, rule)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalSecurityGroupRuleResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state macBareMetalSecurityGroupRuleResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(state.SecurityGroupID.ValueInt64())
	ruleID := int(state.ID.ValueInt64())

	err := r.securityGroupService.Rules(securityGroupID).Delete(ctx, ruleID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete security group rule: %s", err))
		return
	}
}

func (r macBareMetalSecurityGroupRuleResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		validators.MutuallyExclusive("port_range", "icmp"),
	}
}
