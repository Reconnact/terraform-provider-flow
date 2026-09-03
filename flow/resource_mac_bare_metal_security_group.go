package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*macBareMetalSecurityGroupResource)(nil)
	_ resource.ResourceWithConfigure   = (*macBareMetalSecurityGroupResource)(nil)
	_ resource.ResourceWithImportState = (*macBareMetalSecurityGroupResource)(nil)
)

type macBareMetalSecurityGroupResourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	NetworkID types.Int64  `tfsdk:"network_id"`
}

func (r *macBareMetalSecurityGroupResourceData) FromEntity(securityGroup macbaremetal.SecurityGroup) {
	r.ID = types.Int64Value(int64(securityGroup.ID))
	r.Name = types.StringValue(securityGroup.Name)
	r.NetworkID = types.Int64Value(int64(securityGroup.Network.ID))
}

func (r macBareMetalSecurityGroupResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the security group",
				Required:            true,
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func newMacBareMetalSecurityGroupResource() resource.Resource {
	return &macBareMetalSecurityGroupResource{}
}

func (r *macBareMetalSecurityGroupResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mac_bare_metal_security_group"
}

func (r *macBareMetalSecurityGroupResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	r.securityGroupService = macbaremetal.NewSecurityGroupService(client)
}

type macBareMetalSecurityGroupResource struct {
	securityGroupService macbaremetal.SecurityGroupService
}

func (r macBareMetalSecurityGroupResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config macBareMetalSecurityGroupResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := macbaremetal.SecurityGroupCreate{
		Name:        config.Name.ValueString(),
		Description: "a security group created by terraform",
		NetworkID:   int(config.NetworkID.ValueInt64()),
	}

	securityGroup, err := r.securityGroupService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create security group: %s", err))
		return
	}

	var state macBareMetalSecurityGroupResourceData
	state.FromEntity(securityGroup)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalSecurityGroupResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state macBareMetalSecurityGroupResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroup, err := r.securityGroupService.Get(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list security groups: %s", err))
		return
	}

	state.FromEntity(securityGroup)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalSecurityGroupResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state macBareMetalSecurityGroupResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config macBareMetalSecurityGroupResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	update := macbaremetal.SecurityGroupUpdate{
		Name: config.Name.ValueString(),
	}

	securityGroup, err := r.securityGroupService.Update(ctx, int(state.ID.ValueInt64()), update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security group: %s", err))
		return
	}

	state.FromEntity(securityGroup)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalSecurityGroupResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state macBareMetalSecurityGroupResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.securityGroupService.Delete(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete security group: %s", err))
		return
	}
}

func (r macBareMetalSecurityGroupResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}
