package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/common"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeLoadBalancerResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeLoadBalancerResource)(nil)
	_ resource.ResourceWithImportState = (*computeLoadBalancerResource)(nil)
)

type computeLoadBalancerResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
	NetworkID  types.Int64  `tfsdk:"network_id"`
	PrivateIP  types.String `tfsdk:"private_ip"`
}

func (c *computeLoadBalancerResourceData) FromEntity(loadBalancer compute.LoadBalancer) {
	c.ID = types.Int64Value(int64(loadBalancer.ID))
	c.Name = types.StringValue(loadBalancer.Name)
	c.LocationID = types.Int64Value(int64(loadBalancer.Location.ID))

	if len(loadBalancer.Networks) != 0 {
		network := loadBalancer.Networks[0]
		c.NetworkID = types.Int64Value(int64(network.ID))
		c.PrivateIP = types.StringValue(network.Interfaces[0].PrivateIP)
	}
}

func (c computeLoadBalancerResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the load balancer",
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
				MarkdownDescription: "unique identifier of the initial network (the organisation's default network when omitted)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"private_ip": schema.StringAttribute{
				MarkdownDescription: "initial private ip of the load balancer",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func newComputeLoadBalancerResource() resource.Resource {
	return &computeLoadBalancerResource{}
}

func (c *computeLoadBalancerResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_load_balancer"
}

func (c *computeLoadBalancerResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.loadBalancerService = compute.NewLoadBalancerService(client)
	c.orderService = common.NewOrderService(client)
}

type computeLoadBalancerResource struct {
	loadBalancerService compute.LoadBalancerService
	orderService        common.OrderService
}

func (c computeLoadBalancerResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeLoadBalancerResourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.LoadBalancerCreate{
		Name:             config.Name.ValueString(),
		LocationID:       int(config.LocationID.ValueInt64()),
		AttachExternalIP: false,
		NetworkID:        int(config.NetworkID.ValueInt64()),
		PrivateIP:        config.PrivateIP.ValueString(),
	}

	var ordering common.Ordering
	err := retryCreate(ctx, "create load balancer", func() (err error) {
		ordering, err = c.loadBalancerService.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create load balancer: %s", err))
		return
	}

	order, err := waitForOrder(ctx, c.orderService, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for load balancer creation: %s", err))
		return
	}

	loadBalancer, err := waitForLoadBalancerMutable(ctx, c.loadBalancerService, order.Product.ID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for load balancer to be mutable: %s", err))
		if loadBalancer.ID == 0 {
			return
		}
	}

	var state computeLoadBalancerResourceData
	state.FromEntity(loadBalancer)

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeLoadBalancerResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeLoadBalancerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancer, err := c.loadBalancerService.Get(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("load balancer %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get load balancer: %s", err))
		return
	}

	state.FromEntity(loadBalancer)

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeLoadBalancerResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state computeLoadBalancerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeLoadBalancerResourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	update := compute.LoadBalancerUpdate{
		Name: config.Name.ValueString(),
	}

	var loadBalancer compute.LoadBalancer
	err := retry(ctx, "update load balancer", func() (err error) {
		loadBalancer, err = c.loadBalancerService.Update(ctx, int(state.ID.ValueInt64()), update)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update load balancer: %s", err))
		return
	}

	state.FromEntity(loadBalancer)

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeLoadBalancerResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeLoadBalancerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete load balancer", func() error {
		return c.loadBalancerService.Delete(ctx, int(state.ID.ValueInt64()))
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete load balancer: %s", err))
		return
	}
}

func (c computeLoadBalancerResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}

// the api refuses every change to a load balancer, its pools and members while
// the status is working — poll until it settles
func waitForLoadBalancerMutable(ctx context.Context, service compute.LoadBalancerService, loadBalancerID int) (loadBalancer compute.LoadBalancer, err error) {
	err = waitFor(ctx, loadBalancerTimeout, defaultWaitInterval, fmt.Sprintf("load balancer %d to be mutable", loadBalancerID), func(ctx context.Context) (bool, error) {
		got, err := service.Get(ctx, loadBalancerID)
		if err != nil {
			return false, err
		}
		loadBalancer = got

		switch loadBalancer.Status.ID {
		case compute.LoadBalancerStatusWorking:
			return false, nil
		case compute.LoadBalancerStatusError:
			return false, fmt.Errorf("load balancer %d is in error state", loadBalancerID)
		default:
			return true, nil
		}
	})

	return loadBalancer, err
}
