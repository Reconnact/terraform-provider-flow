package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeRouterResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeRouterResource)(nil)
	_ resource.ResourceWithImportState = (*computeRouterResource)(nil)
)

type computeRouterResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
	Public     types.Bool   `tfsdk:"public"`
	PublicIP   types.String `tfsdk:"public_ip"`
}

func (c *computeRouterResourceData) FromEntity(router compute.Router) {
	c.ID = types.Int64Value(int64(router.ID))
	c.Name = types.StringValue(router.Name)
	c.LocationID = types.Int64Value(int64(router.Location.ID))
	c.Public = types.BoolValue(router.Public)

	if router.Public {
		c.PublicIP = types.StringValue(router.PublicIP)
	} else {
		c.PublicIP = types.StringNull()
	}
}

func (c computeRouterResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the router",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the router",
				Required:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"public": schema.BoolAttribute{
				MarkdownDescription: "if the router should be public",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"public_ip": schema.StringAttribute{
				MarkdownDescription: "public IP of the router",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func newComputeRouterResource() resource.Resource {
	return &computeRouterResource{}
}

func (c *computeRouterResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_router"
}

func (c *computeRouterResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.routerService = compute.NewRouterService(client)
}

type computeRouterResource struct {
	routerService compute.RouterService
}

func (c computeRouterResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeRouterResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.RouterCreate{
		Name:       config.Name.ValueString(),
		LocationID: int(config.LocationID.ValueInt64()),
		Public:     true,
	}

	if !config.Public.IsNull() {
		create.Public = config.Public.ValueBool()
	}

	var router compute.Router
	err := retryCreate(ctx, "create router", func() (err error) {
		router, err = c.routerService.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create router: %s", err))
		return
	}

	var state computeRouterResourceData
	state.FromEntity(router)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeRouterResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeRouterResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	router, err := c.routerService.Get(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("router %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get router: %s", err))
		return
	}

	state.FromEntity(router)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeRouterResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state computeRouterResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeRouterResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	update := compute.RouterUpdate{
		Name:   config.Name.ValueString(),
		Public: config.Public.ValueBool(),
	}

	var router compute.Router
	err := retry(ctx, "update router", func() (err error) {
		router, err = c.routerService.Update(ctx, int(state.ID.ValueInt64()), update)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update router: %s", err))
		return
	}

	state.FromEntity(router)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeRouterResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeRouterResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete router", func() error {
		return c.routerService.Delete(ctx, int(state.ID.ValueInt64()))
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete router: %s", err))
		return
	}
}

func (c computeRouterResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}
