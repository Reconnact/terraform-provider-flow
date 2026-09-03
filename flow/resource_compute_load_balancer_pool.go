package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeLoadBalancerPoolResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeLoadBalancerPoolResource)(nil)
	_ resource.ResourceWithImportState = (*computeLoadBalancerPoolResource)(nil)
)

type computeLoadBalancerHTTPHealthCheckResourceData struct {
	Method types.String `tfsdk:"method"`
	Path   types.String `tfsdk:"path"`
}

type computeLoadBalancerHealthCheckResourceData struct {
	TypeID types.Int64                                     `tfsdk:"type_id"`
	HTTP   *computeLoadBalancerHTTPHealthCheckResourceData `tfsdk:"http"`

	Interval types.String `tfsdk:"interval"`
	Timeout  types.String `tfsdk:"timeout"`

	HealthyThreshold   types.Int64 `tfsdk:"healthy_threshold"`
	UnhealthyThreshold types.Int64 `tfsdk:"unhealthy_threshold"`
}

type computeLoadBalancerPoolResourceData struct {
	ID             types.Int64 `tfsdk:"id"`
	LoadBalancerID types.Int64 `tfsdk:"load_balancer_id"`

	Name types.String `tfsdk:"name"`

	BalancingAlgorithmID types.Int64 `tfsdk:"balancing_algorithm_id"`
	StickySession        types.Bool  `tfsdk:"sticky_session"`

	EntryProtocolID  types.Int64 `tfsdk:"entry_protocol_id"`
	EntryPort        types.Int64 `tfsdk:"entry_port"`
	TargetProtocolID types.Int64 `tfsdk:"target_protocol_id"`

	CertificateID types.Int64 `tfsdk:"certificate_id"`

	HealthCheck *computeLoadBalancerHealthCheckResourceData `tfsdk:"health_check"`
}

func (c *computeLoadBalancerPoolResourceData) FromEntity(loadBalancerID int, pool compute.LoadBalancerPool) {
	c.ID = types.Int64Value(int64(pool.ID))
	c.LoadBalancerID = types.Int64Value(int64(loadBalancerID))
	c.Name = types.StringValue(pool.Name)

	c.BalancingAlgorithmID = types.Int64Value(int64(pool.Algorithm.ID))
	c.StickySession = types.BoolValue(pool.StickySession)

	c.EntryProtocolID = types.Int64Value(int64(pool.EntryProtocol.ID))
	c.EntryPort = types.Int64Value(int64(pool.EntryPort))
	c.TargetProtocolID = types.Int64Value(int64(pool.TargetProtocol.ID))

	if pool.Certificate.ID == 0 {
		c.CertificateID = types.Int64Null()
	} else {
		c.CertificateID = types.Int64Value(int64(pool.Certificate.ID))
	}

	c.HealthCheck = &computeLoadBalancerHealthCheckResourceData{
		TypeID:             types.Int64Value(int64(pool.HealthCheck.Type.ID)),
		HTTP:               nil,
		Interval:           types.StringValue((time.Duration(pool.HealthCheck.Interval) * time.Second).String()),
		Timeout:            types.StringValue((time.Duration(pool.HealthCheck.Timeout) * time.Second).String()),
		HealthyThreshold:   types.Int64Value(int64(pool.HealthCheck.HealthyThreshold)),
		UnhealthyThreshold: types.Int64Value(int64(pool.HealthCheck.UnhealthyThreshold)),
	}

	if pool.HealthCheck.HTTPMethod != "" || pool.HealthCheck.HTTPPath != "" {
		c.HealthCheck.HTTP = &computeLoadBalancerHTTPHealthCheckResourceData{
			Method: types.StringValue(pool.HealthCheck.HTTPMethod),
			Path:   types.StringValue(pool.HealthCheck.HTTPPath),
		}
	}
}

func (c computeLoadBalancerPoolResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Import: `terraform import flow_compute_load_balancer_pool.<name> <load_balancer_id>:<id>`",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer pool",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"load_balancer_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "name of the load balancer pool",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"balancing_algorithm_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the balancing algorithm",
				Required:            true,
			},
			"sticky_session": schema.BoolAttribute{
				MarkdownDescription: "whether the load balancer pool is sticky",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},

			"entry_protocol_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the entry protocol",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"entry_port": schema.Int64Attribute{
				MarkdownDescription: "entry port of the load balancer pool",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"target_protocol_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the target protocol",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},

			"certificate_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the certificate",
				Optional:            true,
			},

			"health_check": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"type_id": schema.Int64Attribute{
						MarkdownDescription: "unique identifier of the health check type",
						Required:            true,
					},
					"http": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"method": schema.StringAttribute{
								MarkdownDescription: "HTTP method of the health check",
								Required:            true,
							},
							"path": schema.StringAttribute{
								MarkdownDescription: "path of the health check",
								Required:            true,
							},
						},
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"interval": schema.StringAttribute{
						MarkdownDescription: "interval duration of the health check",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"timeout": schema.StringAttribute{
						MarkdownDescription: "timeout duration of the health check",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"healthy_threshold": schema.Int64Attribute{
						MarkdownDescription: "number of successful health checks before considering the target healthy",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"unhealthy_threshold": schema.Int64Attribute{
						MarkdownDescription: "number of failed health checks before considering the target unhealthy",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
				Required: true,
			},
		},
	}
}

func newComputeLoadBalancerPoolResource() resource.Resource {
	return &computeLoadBalancerPoolResource{}
}

func (c *computeLoadBalancerPoolResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_load_balancer_pool"
}

func (c *computeLoadBalancerPoolResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.loadBalancerService = compute.NewLoadBalancerService(client)
}

type computeLoadBalancerPoolResource struct {
	loadBalancerService compute.LoadBalancerService
}

func (c computeLoadBalancerPoolResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeLoadBalancerPoolResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	healthCheck, diagnostics := convertHealthCheckConfigToAPIOptions(*config.HealthCheck)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(config.LoadBalancerID.ValueInt64())

	create := compute.LoadBalancerPoolCreate{
		EntryProtocolID:      int(config.EntryProtocolID.ValueInt64()),
		TargetProtocolID:     int(config.TargetProtocolID.ValueInt64()),
		CertificateID:        int(config.CertificateID.ValueInt64()),
		EntryPort:            int(config.EntryPort.ValueInt64()),
		BalancingAlgorithmID: int(config.BalancingAlgorithmID.ValueInt64()),
		StickySession:        config.StickySession.ValueBool(),
		HealthCheck:          healthCheck,
	}

	var pool compute.LoadBalancerPool
	err := retryCreate(ctx, "create load balancer pool", func() (err error) {
		pool, err = c.loadBalancerService.Pools(loadBalancerID).Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create load balancer pool: %s", err))
		return
	}

	var state computeLoadBalancerPoolResourceData
	state.FromEntity(loadBalancerID, pool)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)

	_, err = waitForLoadBalancerMutable(ctx, c.loadBalancerService, loadBalancerID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for load balancer to be mutable: %s", err))
	}
}

func (c computeLoadBalancerPoolResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeLoadBalancerPoolResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(state.LoadBalancerID.ValueInt64())

	pool, err := c.loadBalancerService.Pools(loadBalancerID).Get(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("load balancer pool %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get load balancer pool: %s", err))
		return
	}

	state.FromEntity(loadBalancerID, pool)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeLoadBalancerPoolResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state computeLoadBalancerPoolResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeLoadBalancerPoolResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	healthCheck, diagnostics := convertHealthCheckConfigToAPIOptions(*config.HealthCheck)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(state.LoadBalancerID.ValueInt64())
	poolID := int(state.ID.ValueInt64())

	update := compute.LoadBalancerPoolUpdate{
		CertificateID:        int(config.CertificateID.ValueInt64()),
		BalancingAlgorithmID: int(config.BalancingAlgorithmID.ValueInt64()),
		StickySession:        config.StickySession.ValueBool(),
		HealthCheck:          healthCheck,
	}

	var pool compute.LoadBalancerPool
	err := retry(ctx, "update load balancer pool", func() (err error) {
		pool, err = c.loadBalancerService.Pools(loadBalancerID).Update(ctx, poolID, update)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update load balancer pool: %s", err))
		return
	}

	_, err = waitForLoadBalancerMutable(ctx, c.loadBalancerService, loadBalancerID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for load balancer to be mutable: %s", err))
		return
	}

	state.FromEntity(loadBalancerID, pool)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeLoadBalancerPoolResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeLoadBalancerPoolResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(state.LoadBalancerID.ValueInt64())
	poolID := int(state.ID.ValueInt64())

	err := retryDelete(ctx, "delete load balancer pool", func() error {
		return c.loadBalancerService.Pools(loadBalancerID).Delete(ctx, poolID)
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete load balancer pool: %s", err))
		return
	}

	_, err = waitForLoadBalancerMutable(ctx, c.loadBalancerService, loadBalancerID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for load balancer to be mutable: %s", err))
		return
	}
}

func convertHealthCheckConfigToAPIOptions(config computeLoadBalancerHealthCheckResourceData) (options compute.LoadBalancerHealthCheckOptions, diagnostics diag.Diagnostics) {
	healthCheckIntervalSeconds := 0
	healthCheckTimeoutSeconds := 0

	if !config.Interval.IsNull() {
		duration, err := time.ParseDuration(config.Interval.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid Interval", fmt.Sprintf("unable to parse health check interval: %s", err))
			return
		}

		healthCheckIntervalSeconds = int(duration.Milliseconds() / 1000)
	}

	if !config.Timeout.IsNull() {
		duration, err := time.ParseDuration(config.Timeout.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid Timeout", fmt.Sprintf("unable to parse health check timeout: %s", err))
			return
		}

		healthCheckTimeoutSeconds = int(duration.Milliseconds() / 1000)
	}

	options = compute.LoadBalancerHealthCheckOptions{
		TypeID:             int(config.TypeID.ValueInt64()),
		Interval:           healthCheckIntervalSeconds,
		Timeout:            healthCheckTimeoutSeconds,
		HealthyThreshold:   int(config.HealthyThreshold.ValueInt64()),
		UnhealthyThreshold: int(config.UnhealthyThreshold.ValueInt64()),
	}

	if config.HTTP != nil {
		options.HTTPMethod = config.HTTP.Method.ValueString()
		options.HTTPPath = config.HTTP.Path.ValueString()
	}

	return
}

func (c computeLoadBalancerPoolResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStateCompositeInt64IDs(ctx, request, response, path.Root("load_balancer_id"), path.Root("id"))
}
