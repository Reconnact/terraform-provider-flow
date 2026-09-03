package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*macBareMetalElasticIPDeviceAttachmentResource)(nil)
	_ resource.ResourceWithConfigure   = (*macBareMetalElasticIPDeviceAttachmentResource)(nil)
	_ resource.ResourceWithImportState = (*macBareMetalElasticIPDeviceAttachmentResource)(nil)
)

type macBareMetalElasticIPDeviceAttachmentResourceData struct {
	DeviceID           types.Int64 `tfsdk:"device_id"`
	NetworkInterfaceID types.Int64 `tfsdk:"network_interface_id"`
	ElasticIPID        types.Int64 `tfsdk:"elastic_ip_id"`
}

func (m *macBareMetalElasticIPDeviceAttachmentResourceData) FromEntity(
	device macbaremetal.Device,
	elasticIP macbaremetal.ElasticIP,
) {
	m.DeviceID = types.Int64Value(int64(device.ID))
	m.NetworkInterfaceID = types.Int64Null()
	m.ElasticIPID = types.Int64Value(int64(elasticIP.ID))

	for _, iface := range device.NetworkInterfaces {
		if iface.PublicIP == elasticIP.PublicIP {
			m.NetworkInterfaceID = types.Int64Value(int64(iface.ID))
		}
	}
}

func (m macBareMetalElasticIPDeviceAttachmentResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Import: `terraform import flow_mac_bare_metal_elastic_ip_device_attachment.<name> <device_id>:<elastic_ip_id>`",
		Attributes: map[string]schema.Attribute{
			"device_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the device to attach the elastic ip to",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"network_interface_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network interface of the device to attach the elastic ip to",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"elastic_ip_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the elastic ip to attach to the device",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func newMacBareMetalElasticIPDeviceAttachmentResource() resource.Resource {
	return &macBareMetalElasticIPDeviceAttachmentResource{}
}

func (c *macBareMetalElasticIPDeviceAttachmentResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mac_bare_metal_elastic_ip_attachment"
}

func (c *macBareMetalElasticIPDeviceAttachmentResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.deviceService = macbaremetal.NewDeviceService(client)
	c.elasticIPService = macbaremetal.NewElasticIPService(client)
	c.client = client
}

type macBareMetalElasticIPDeviceAttachmentResource struct {
	deviceService    macbaremetal.DeviceService
	elasticIPService macbaremetal.ElasticIPService

	client goclient.Client
}

func (c macBareMetalElasticIPDeviceAttachmentResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config macBareMetalElasticIPDeviceAttachmentResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(config.DeviceID.ValueInt64())

	attach := macbaremetal.ElasticIPAttach{
		ElasticIPID:        int(config.ElasticIPID.ValueInt64()),
		NetworkInterfaceID: int(config.NetworkInterfaceID.ValueInt64()),
	}

	elasticIP, err := macbaremetal.NewAttachedElasticIPService(c.client, serverID).Attach(ctx, attach)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to attach elastic ip: %s", err))
		return
	}

	device, err := macbaremetal.NewDeviceService(c.client).Get(ctx, serverID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get device: %s", err))
		return
	}

	var state macBareMetalElasticIPDeviceAttachmentResourceData
	state.FromEntity(device, elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c macBareMetalElasticIPDeviceAttachmentResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state macBareMetalElasticIPDeviceAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	device, err := macbaremetal.NewDeviceService(c.client).Get(ctx, int(state.DeviceID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get device: %s", err))
		return
	}

	elasticIP, diagnostics := findMacBareMetalElasticIP(ctx, c.elasticIPService, int(state.ElasticIPID.ValueInt64()))
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	state.FromEntity(device, elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c macBareMetalElasticIPDeviceAttachmentResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	response.Diagnostics.AddError("Not Supported", "updating an elastic ip attachment is not supported")
}

func (c macBareMetalElasticIPDeviceAttachmentResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state macBareMetalElasticIPDeviceAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := macbaremetal.NewAttachedElasticIPService(c.client, int(state.DeviceID.ValueInt64())).Detach(ctx, int(state.ElasticIPID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to detach elastic ip: %s", err))
		return
	}
}

func (c macBareMetalElasticIPDeviceAttachmentResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStateCompositeInt64IDs(ctx, request, response, path.Root("device_id"), path.Root("elastic_ip_id"))
}
