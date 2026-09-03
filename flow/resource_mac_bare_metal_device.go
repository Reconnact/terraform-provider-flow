package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/common"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*macBareMetalDeviceResource)(nil)
	_ resource.ResourceWithConfigure   = (*macBareMetalDeviceResource)(nil)
	_ resource.ResourceWithImportState = (*macBareMetalDeviceResource)(nil)
)

type macBareMetalDeviceResourceData struct {
	ID                 types.Int64  `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	LocationID         types.Int64  `tfsdk:"location_id"`
	ProductID          types.Int64  `tfsdk:"product_id"`
	NetworkID          types.Int64  `tfsdk:"network_id"`
	NetworkInterfaceID types.Int64  `tfsdk:"network_interface_id"`
	Password           types.String `tfsdk:"password"`
}

func (m *macBareMetalDeviceResourceData) FromEntity(device macbaremetal.Device) {
	m.ID = types.Int64Value(int64(device.ID))
	m.Name = types.StringValue(device.Name)
	m.LocationID = types.Int64Value(int64(device.Location.ID))
	m.ProductID = types.Int64Value(int64(device.Product.ID))
	m.NetworkID = types.Int64Value(int64(device.Network.ID))

	if len(device.NetworkInterfaces) > 0 {
		m.NetworkInterfaceID = types.Int64Value(int64(device.NetworkInterfaces[0].ID))
	}
}

func (m macBareMetalDeviceResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the device",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the device",
				Required:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"network_interface_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network interface",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"product_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the product",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "password of the device",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func newMacBareMetalDeviceResource() resource.Resource {
	return &macBareMetalDeviceResource{}
}

func (m *macBareMetalDeviceResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mac_bare_metal_device"
}

func (m *macBareMetalDeviceResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	m.orderService = common.NewOrderService(client)
	m.deviceService = macbaremetal.NewDeviceService(client)
}

type macBareMetalDeviceResource struct {
	orderService  common.OrderService
	deviceService macbaremetal.DeviceService
}

func (m macBareMetalDeviceResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config macBareMetalDeviceResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := macbaremetal.DeviceCreate{
		Name:            config.Name.ValueString(),
		LocationID:      int(config.LocationID.ValueInt64()),
		ProductID:       int(config.ProductID.ValueInt64()),
		NetworkID:       int(config.NetworkID.ValueInt64()),
		AttachElasticIP: false,
		Password:        config.Password.ValueString(),
	}

	ordering, err := m.deviceService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create device: %s", err))
		return
	}

	order, err := waitForOrder(ctx, m.orderService, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for device creation: %s", err))
		return
	}

	device, err := m.deviceService.Get(ctx, order.Product.ID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get device: %s", err))
		return
	}

	var state macBareMetalDeviceResourceData
	state.FromEntity(device)

	state.Password = config.Password

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (m macBareMetalDeviceResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state macBareMetalDeviceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	device, err := m.deviceService.Get(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get device: %s", err))
		return
	}

	state.FromEntity(device)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (m macBareMetalDeviceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state macBareMetalDeviceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config macBareMetalDeviceResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	update := macbaremetal.DeviceUpdate{
		Name: config.Name.ValueString(),
	}

	device, err := m.deviceService.Update(ctx, int(state.ID.ValueInt64()), update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update device: %s", err))
		return
	}

	state.FromEntity(device)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (m macBareMetalDeviceResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state macBareMetalDeviceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := m.deviceService.Delete(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete device: %s", err))
		return
	}
}

func (m macBareMetalDeviceResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}
