package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/common"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeServerResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeServerResource)(nil)
	_ resource.ResourceWithImportState = (*computeServerResource)(nil)
)

type computeServerResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
	ImageID    types.Int64  `tfsdk:"image_id"`
	ProductID  types.Int64  `tfsdk:"product_id"`
	NetworkID  types.Int64  `tfsdk:"network_id"`
	PrivateIP  types.String `tfsdk:"private_ip"`
	KeyPairID  types.Int64  `tfsdk:"key_pair_id"`
	Password   types.String `tfsdk:"password"`
	CloudInit  types.String `tfsdk:"cloud_init"`

	NetworkInterfaceID types.Int64 `tfsdk:"network_interface_id"`
	SecurityGroupIDs   types.Set   `tfsdk:"security_group_ids"`
}

func (c *computeServerResourceData) FromEntity(server compute.Server) {
	c.ID = types.Int64Value(int64(server.ID))
	c.Name = types.StringValue(server.Name)
	c.LocationID = types.Int64Value(int64(server.Location.ID))
	c.ImageID = types.Int64Value(int64(server.Image.ID))
	c.ProductID = types.Int64Value(int64(server.Product.ID))
	c.KeyPairID = types.Int64Null()
	if server.KeyPair.ID != 0 {
		c.KeyPairID = types.Int64Value(int64(server.KeyPair.ID))
	}

	c.NetworkInterfaceID = types.Int64Null()
	if len(server.Networks) != 0 {
		network := server.Networks[0]
		c.NetworkID = types.Int64Value(int64(network.ID))
		if len(network.Interfaces) != 0 {
			c.PrivateIP = types.StringValue(network.Interfaces[0].PrivateIP)
			c.NetworkInterfaceID = types.Int64Value(int64(network.Interfaces[0].ID))
		}
	}
}

func primaryInterfaceID(server compute.Server) (int, bool) {
	if len(server.Networks) == 0 || len(server.Networks[0].Interfaces) == 0 {
		return 0, false
	}
	return server.Networks[0].Interfaces[0].ID, true
}

func (c computeServerResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the server",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the server",
				Required:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"image_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the image",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"product_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the product — changing it resizes the server in place: it is stopped, resized and started again (about a minute of downtime), disks and addresses are kept",
				Required:            true,
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the initial network (the organisation's default network when omitted)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"private_ip": schema.StringAttribute{
				MarkdownDescription: "initial private ip of the server",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"network_interface_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the server's primary network interface — reference it from elastic ip attachments",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"security_group_ids": schema.SetAttribute{
				ElementType:         types.Int64Type,
				MarkdownDescription: "security groups on the primary network interface — the organisation's default group when omitted; at least one is required",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"key_pair_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the key pair (linux images require one)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "initial windows password of the server",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					// TODO: write-only once the framework is on 1.x (Terraform ≥ 1.11 WriteOnly attributes) — until then an imported resource plans a replace here because the api never returns the value
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cloud_init": schema.StringAttribute{
				MarkdownDescription: "cloud init script",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					// TODO: write-only once the framework is on 1.x (Terraform ≥ 1.11 WriteOnly attributes) — until then an imported resource plans a replace here because the api never returns the value
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func newComputeServerResource() resource.Resource {
	return &computeServerResource{}
}

func (c *computeServerResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_server"
}

func (c *computeServerResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.serverService = compute.NewServerService(client)
	c.orderService = common.NewOrderService(client)
}

type computeServerResource struct {
	serverService compute.ServerService
	orderService  common.OrderService
}

func (c computeServerResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeServerResourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.ServerCreate{
		Name:             config.Name.ValueString(),
		LocationID:       int(config.LocationID.ValueInt64()),
		ImageID:          int(config.ImageID.ValueInt64()),
		ProductID:        int(config.ProductID.ValueInt64()),
		AttachExternalIP: false,
		NetworkID:        int(config.NetworkID.ValueInt64()),
		PrivateIP:        config.PrivateIP.ValueString(),
		KeyPairID:        int(config.KeyPairID.ValueInt64()),
		Password:         config.Password.ValueString(),
		CloudInit:        config.CloudInit.ValueString(),
	}

	var ordering common.Ordering
	err := retryCreate(ctx, "create server", func() (err error) {
		ordering, err = c.serverService.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create server: %s", err))
		return
	}

	order, err := waitForOrder(ctx, c.orderService, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for server creation: %s", err))
		return
	}

	server, err := c.waitForServerStatus(ctx, order.Product.ID, compute.ServerStatusRunning, "running")
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for server to be running: %s", err))
		if server.ID == 0 {
			return
		}
	} else if !config.SecurityGroupIDs.IsNull() && !config.SecurityGroupIDs.IsUnknown() {
		if err := c.updateSecurityGroups(ctx, server, config.SecurityGroupIDs); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security groups: %s", err))
		}
	}

	var state computeServerResourceData
	state.FromEntity(server)

	state.Password = config.Password
	state.CloudInit = config.CloudInit

	response.Diagnostics.Append(c.readSecurityGroups(ctx, server, &state)...)
	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeServerResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeServerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	server, err := c.serverService.Get(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("server %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get server: %s", err))
		return
	}

	state.FromEntity(server)

	response.Diagnostics.Append(c.readSecurityGroups(ctx, server, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeServerResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state computeServerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	var plan computeServerResourceData
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	update := compute.ServerUpdate{
		Name: plan.Name.ValueString(),
	}

	var server compute.Server
	err := retry(ctx, "update server", func() (err error) {
		server, err = c.serverService.Update(ctx, int(state.ID.ValueInt64()), update)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update server: %s", err))
		return
	}

	if plan.ProductID.ValueInt64() != state.ProductID.ValueInt64() {
		resized, err := c.resize(ctx, server, int(plan.ProductID.ValueInt64()))
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to resize server: %s", err))
			return
		}
		server = resized
	}

	if !plan.SecurityGroupIDs.IsUnknown() && !plan.SecurityGroupIDs.IsNull() && !plan.SecurityGroupIDs.Equal(state.SecurityGroupIDs) {
		if err := c.updateSecurityGroups(ctx, server, plan.SecurityGroupIDs); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security groups: %s", err))
			return
		}
	}

	state.FromEntity(server)

	response.Diagnostics.Append(c.readSecurityGroups(ctx, server, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeServerResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeServerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete server", func() error {
		return c.serverService.Delete(ctx, int(state.ID.ValueInt64()), false)
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete server: %s", err))
		return
	}
}

func (c computeServerResource) waitForServerStatus(ctx context.Context, serverID int, want int, name string) (server compute.Server, err error) {
	err = waitFor(ctx, serverBootTimeout, defaultWaitInterval, fmt.Sprintf("server %d to be %s", serverID, name), func(ctx context.Context) (bool, error) {
		got, err := c.serverService.Get(ctx, serverID)
		if err != nil {
			return false, err
		}
		server = got

		switch server.Status.ID {
		case want:
			return true, nil
		case compute.ServerStatusError:
			return false, fmt.Errorf("server %d is in error state", serverID)
		default:
			return false, nil
		}
	})

	return server, err
}

func (c computeServerResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}

func (c computeServerResource) updateSecurityGroups(ctx context.Context, server compute.Server, groups types.Set) error {
	ifaceID, ok := primaryInterfaceID(server)
	if !ok {
		return fmt.Errorf("server %d has no network interface", server.ID)
	}

	update := compute.NetworkInterfaceSecurityGroupUpdate{SecurityGroupIDs: securityGroupIDs(groups)}
	return retry(ctx, "update security groups", func() (err error) {
		_, err = c.serverService.NetworkInterfaces(server.ID).UpdateSecurityGroups(ctx, ifaceID, update)
		return err
	})
}

func (c computeServerResource) readSecurityGroups(ctx context.Context, server compute.Server, state *computeServerResourceData) (diagnostics diag.Diagnostics) {
	state.SecurityGroupIDs = types.SetNull(types.Int64Type)

	ifaceID, ok := primaryInterfaceID(server)
	if !ok {
		return nil
	}

	list, err := c.serverService.NetworkInterfaces(server.ID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("unable to list network interfaces of server %d: %s", server.ID, err))
		return diagnostics
	}

	for _, iface := range list.Items {
		if iface.ID == ifaceID {
			state.SecurityGroupIDs = securityGroupIDSet(iface)
			break
		}
	}
	return nil
}

const (
	serverActionStart = "start"
	serverActionStop  = "stop"
)

// the api resizes only a stopped server: stop, upgrade (an order), start —
// a server the user keeps stopped stays stopped
func (c computeServerResource) resize(ctx context.Context, server compute.Server, productID int) (compute.Server, error) {
	wasStopped := server.Status.ID == compute.ServerStatusStopped
	if !wasStopped {
		if err := c.perform(ctx, server.ID, serverActionStop); err != nil {
			return server, err
		}
		if _, err := c.waitForServerStatus(ctx, server.ID, compute.ServerStatusStopped, "stopped"); err != nil {
			return server, err
		}
	}

	var ordering common.Ordering
	err := retry(ctx, "upgrade server", func() (err error) {
		ordering, err = c.serverService.Upgrade(ctx, server.ID, compute.ServerUpgrade{ProductID: productID})
		return err
	})
	if err == nil {
		_, err = waitForOrder(ctx, c.orderService, ordering)
	}
	if err == nil {
		_, err = c.waitForServerStatus(ctx, server.ID, compute.ServerStatusStopped, "stopped")
	}

	if wasStopped {
		if err != nil {
			return server, err
		}
		return c.serverService.Get(ctx, server.ID)
	}

	// started again even after a failed upgrade — a resize must not leave a
	// stopped server behind
	if startErr := c.perform(ctx, server.ID, serverActionStart); startErr != nil && err == nil {
		err = startErr
	}
	running, waitErr := c.waitForServerStatus(ctx, server.ID, compute.ServerStatusRunning, "running")
	if err != nil {
		return running, err
	}
	return running, waitErr
}

func (c computeServerResource) perform(ctx context.Context, serverID int, action string) error {
	return retry(ctx, action+" server", func() (err error) {
		_, err = c.serverService.Perform(ctx, serverID, compute.ServerPerform{Action: action})
		return err
	})
}
