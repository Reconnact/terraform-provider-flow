package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*macBareMetalNetworkResource)(nil)
	_ resource.ResourceWithConfigure   = (*macBareMetalNetworkResource)(nil)
	_ resource.ResourceWithImportState = (*macBareMetalNetworkResource)(nil)
)

type macBareMetalNetworkResourceAllocationPool struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type macBareMetalNetworkResourceData struct {
	ID                types.Int64                                `tfsdk:"id"`
	Name              types.String                               `tfsdk:"name"`
	CIDR              types.String                               `tfsdk:"cidr"`
	LocationID        types.Int64                                `tfsdk:"location_id"`
	DomainName        types.String                               `tfsdk:"domain_name"`
	DomainNameServers []types.String                             `tfsdk:"domain_name_servers"`
	AllocationPool    *macBareMetalNetworkResourceAllocationPool `tfsdk:"allocation_pool"`
	GatewayIP         types.String                               `tfsdk:"gateway_ip"`
}

func (r *macBareMetalNetworkResourceData) FromEntity(network macbaremetal.Network) {
	r.ID = types.Int64Value(int64(network.ID))
	r.Name = types.StringValue(network.Name)
	r.CIDR = types.StringValue(network.Subnet)
	r.LocationID = types.Int64Value(int64(network.Location.ID))
	r.GatewayIP = types.StringValue(network.GatewayIP)

	r.AllocationPool = &macBareMetalNetworkResourceAllocationPool{
		Start: types.StringValue(network.AllocationPoolStart),
		End:   types.StringValue(network.AllocationPoolEnd),
	}

	r.DomainName = types.StringValue(network.DomainName)
	r.DomainNameServers = make([]types.String, len(network.DomainNameServers))
	for idx, domainNameServer := range network.DomainNameServers {
		r.DomainNameServers[idx] = types.StringValue(domainNameServer)
	}
}

func (r macBareMetalNetworkResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the network",
				Required:            true,
			},
			"cidr": schema.StringAttribute{
				MarkdownDescription: "CIDR of the network",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"domain_name": schema.StringAttribute{
				MarkdownDescription: "domain name of the network",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_name_servers": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "list of domain name servers",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"gateway_ip": schema.StringAttribute{
				MarkdownDescription: "gateway IP of the network",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func newMacBareMetalNetworkResource() resource.Resource {
	return &macBareMetalNetworkResource{}
}

func (r *macBareMetalNetworkResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mac_bare_metal_network"
}

func (r *macBareMetalNetworkResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	r.networkService = macbaremetal.NewNetworkService(client)
}

type macBareMetalNetworkResource struct {
	networkService macbaremetal.NetworkService
}

func (r macBareMetalNetworkResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config macBareMetalNetworkResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := macbaremetal.NetworkCreate{
		Name:       config.Name.ValueString(),
		LocationID: int(config.LocationID.ValueInt64()),
	}

	network, err := r.networkService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create network: %s", err))
		return
	}

	// the api does not allow to set these properties on creation,
	// so we need to set afterwards using an update.
	if len(config.DomainNameServers) != 0 || !config.DomainName.IsNull() {
		update := macbaremetal.NetworkUpdate{
			DomainName:        config.DomainName.ValueString(),
			DomainNameServers: nil,
		}

		for _, domainNameServer := range config.DomainNameServers {
			update.DomainNameServers = append(update.DomainNameServers, domainNameServer.ValueString())
		}

		network, err = r.networkService.Update(ctx, network.ID, update)
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update network: %s", err))
			return
		}
	}

	var state macBareMetalNetworkResourceData
	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalNetworkResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state macBareMetalNetworkResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	network, err := r.networkService.Get(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get network: %s", err))
		return
	}

	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalNetworkResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state macBareMetalNetworkResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config macBareMetalNetworkResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	update := macbaremetal.NetworkUpdate{
		Name:       config.Name.ValueString(),
		DomainName: config.DomainName.ValueString(),
	}

	if len(config.DomainNameServers) != 0 {
		update.DomainNameServers = make([]string, len(config.DomainNameServers))
		for idx, domainNameServer := range config.DomainNameServers {
			update.DomainNameServers[idx] = domainNameServer.ValueString()
		}
	}

	network, err := r.networkService.Update(ctx, int(state.ID.ValueInt64()), update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update network: %s", err))
		return
	}

	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalNetworkResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state macBareMetalNetworkResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.networkService.Delete(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete network: %s", err))
		return
	}
}

func (r macBareMetalNetworkResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}
