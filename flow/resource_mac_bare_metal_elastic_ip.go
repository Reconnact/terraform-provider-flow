package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*macBareMetalElasticIPResource)(nil)
	_ resource.ResourceWithConfigure   = (*macBareMetalElasticIPResource)(nil)
	_ resource.ResourceWithImportState = (*macBareMetalElasticIPResource)(nil)
)

type macBareMetalElasticIPResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	LocationID types.Int64  `tfsdk:"location_id"`
	PublicIP   types.String `tfsdk:"public_ip"`
}

func (r *macBareMetalElasticIPResourceData) FromEntity(elasticIP macbaremetal.ElasticIP) {
	r.ID = types.Int64Value(int64(elasticIP.ID))
	r.LocationID = types.Int64Value(int64(elasticIP.Location.ID))
	r.PublicIP = types.StringValue(elasticIP.PublicIP)
}

func (r macBareMetalElasticIPResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the elastic ip",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "location of the elastic ip",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"public_ip": schema.StringAttribute{
				MarkdownDescription: "public ip address",
				Computed:            true,
			},
		},
	}
}

func newMacBareMetalElasticIPResource() resource.Resource {
	return &macBareMetalElasticIPResource{}
}

func (r *macBareMetalElasticIPResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mac_bare_metal_elastic_ip"
}

func (r *macBareMetalElasticIPResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	r.elasticIPService = macbaremetal.NewElasticIPService(client)
}

type macBareMetalElasticIPResource struct {
	elasticIPService macbaremetal.ElasticIPService
}

func (r macBareMetalElasticIPResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config macBareMetalElasticIPResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := macbaremetal.ElasticIPCreate{
		LocationID: int(config.LocationID.ValueInt64()),
	}

	elasticIP, err := r.elasticIPService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create elastic ip: %s", err))
		return
	}

	tflog.Trace(ctx, "created elastic ip", map[string]interface{}{
		"id":   elasticIP.ID,
		"data": elasticIP,
	})

	var state macBareMetalElasticIPResourceData
	state.FromEntity(elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalElasticIPResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state macBareMetalElasticIPResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	elasticIP, diagnostics := findMacBareMetalElasticIP(ctx, r.elasticIPService, int(state.ID.ValueInt64()))
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	state.FromEntity(elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalElasticIPResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	response.Diagnostics.AddError("Not Supported", "updating an elastic ip is not supported")
}

func (r macBareMetalElasticIPResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state macBareMetalElasticIPResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.elasticIPService.Delete(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete elastic ip: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted elastic ip", map[string]interface{}{
		"id": int(state.ID.ValueInt64()),
	})
}

func (r macBareMetalElasticIPResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}

func findMacBareMetalElasticIP(ctx context.Context, service macbaremetal.ElasticIPService, id int) (elasticIP macbaremetal.ElasticIP, diagnostics diag.Diagnostics) {
	list, err := service.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("unable to list elastic ips: %s", err))
		return
	}

	for _, elasticIP = range list.Items {
		if elasticIP.ID == id {
			return
		}
	}

	diagnostics.AddError("Not Found", fmt.Sprintf("unable to find elastic ip with id %d", id))
	return
}
