package flow

import (
	"context"
	"fmt"
	"net/http"

	"github.com/flowswiss/goclient/common"
	"github.com/flowswiss/goclient/compute"
	"github.com/flowswiss/goclient/kubernetes"
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
	_ resource.Resource                = (*kubernetesClusterResource)(nil)
	_ resource.ResourceWithConfigure   = (*kubernetesClusterResource)(nil)
	_ resource.ResourceWithImportState = (*kubernetesClusterResource)(nil)
)

type kubernetesClusterResourceData struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	LocationID      types.Int64 `tfsdk:"location_id"`
	NetworkID       types.Int64 `tfsdk:"network_id"`
	SecurityGroupID types.Int64 `tfsdk:"security_group_id"`

	Public        types.Bool   `tfsdk:"public"`
	PublicAddress types.String `tfsdk:"public_address"`
	DNSName       types.String `tfsdk:"dns_name"`

	VersionID types.Int64 `tfsdk:"version_id"`

	NodeCount     types.Int64 `tfsdk:"node_count"`
	NodeProductID types.Int64 `tfsdk:"node_product_id"`
}

func (k *kubernetesClusterResourceData) FromEntity(cluster kubernetes.Cluster) {
	k.ID = types.Int64Value(int64(cluster.ID))
	k.Name = types.StringValue(cluster.Name)

	k.LocationID = types.Int64Value(int64(cluster.Location.ID))
	k.NetworkID = types.Int64Value(int64(cluster.Network.ID))
	k.SecurityGroupID = types.Int64Value(int64(cluster.SecurityGroup.ID))

	if cluster.PublicAddress == "" {
		k.Public = types.BoolValue(false)
		k.PublicAddress = types.StringNull()
	} else {
		k.Public = types.BoolValue(true)
		k.PublicAddress = types.StringValue(cluster.PublicAddress)
	}

	k.DNSName = types.StringValue(cluster.DNSName)

	k.VersionID = types.Int64Value(int64(cluster.Version.ID))

	k.NodeCount = types.Int64Value(int64(cluster.NodeCount.Expected.Worker))
	k.NodeProductID = types.Int64Value(int64(cluster.ExpectedPreset.Worker.ID))
}

type kubernetesClusterNameFilter struct {
	Name string
}

func (f kubernetesClusterNameFilter) AppliesTo(cluster kubernetes.Cluster) bool {
	return cluster.Name == f.Name
}

func (k kubernetesClusterResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the cluster",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the cluster",
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
					int64planmodifier.RequiresReplace(),
				},
			},
			"security_group_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"public": schema.BoolAttribute{
				MarkdownDescription: "indicates if the cluster is public",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"public_address": schema.StringAttribute{
				MarkdownDescription: "public address of the cluster",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dns_name": schema.StringAttribute{
				MarkdownDescription: "DNS name of the cluster",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the kubernetes version",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"node_count": schema.Int64Attribute{
				MarkdownDescription: "number of nodes in the cluster",
				Required:            true,
			},
			"node_product_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the node product",
				Required:            true,
			},
		},
	}
}

func newKubernetesClusterResource() resource.Resource {
	return &kubernetesClusterResource{}
}

func (k *kubernetesClusterResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kubernetes_cluster"
}

func (k *kubernetesClusterResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	k.orderService = common.NewOrderService(client)
	k.clusterService = kubernetes.NewClusterService(client)
}

type kubernetesClusterResource struct {
	orderService   common.OrderService
	clusterService kubernetes.ClusterService
}

func (k kubernetesClusterResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config kubernetesClusterResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := kubernetes.ClusterCreate{
		Name:       config.Name.ValueString(),
		LocationID: int(config.LocationID.ValueInt64()),
		NetworkID:  int(config.NetworkID.ValueInt64()),
		Worker: kubernetes.ClusterWorkerCreate{
			ProductID: int(config.NodeProductID.ValueInt64()),
			Count:     int(config.NodeCount.ValueInt64()),
		},
		AttachExternalIP: true,
	}

	if !config.Public.IsNull() && !config.Public.ValueBool() {
		create.AttachExternalIP = false
	}

	var ordering common.Ordering
	err := retryCreate(ctx, "create cluster", func() (err error) {
		ordering, err = k.clusterService.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create cluster: %s", err))
		return
	}

	order, err := waitForOrder(ctx, k.orderService, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster creation: %s", err))
		return
	}

	cluster, err := waitForClusterReady(ctx, k.clusterService, order.Product.ID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be ready: %s", err))
		if cluster.ID == 0 {
			return
		}
	}

	// set state of the resource
	var state kubernetesClusterResourceData
	state.FromEntity(cluster)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (k kubernetesClusterResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state kubernetesClusterResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	cluster, err := k.clusterService.Get(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("cluster %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get cluster: %s", err))
		return
	}

	state.FromEntity(cluster)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (k kubernetesClusterResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state kubernetesClusterResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var plan kubernetesClusterResourceData
	diagnostics = request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	if plan.Name.ValueString() != state.Name.ValueString() {
		// no unlock wait here — the name update is not guarded by the action , unlike configuration and flavor
		update := kubernetes.ClusterUpdate{
			Name: plan.Name.ValueString(),
		}

		err := retry(ctx, "update cluster", func() (err error) {
			_, err = k.clusterService.Update(ctx, int(state.ID.ValueInt64()), update)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update cluster: %s", err))
			return
		}
	}

	if !plan.VersionID.IsUnknown() && plan.VersionID.ValueInt64() != state.VersionID.ValueInt64() {
		if _, err := k.waitForClusterUnlocked(ctx, int(state.ID.ValueInt64())); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be unlocked: %s", err))
			return
		}

		current, err := k.clusterService.GetConfiguration(ctx, int(state.ID.ValueInt64()))
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to read cluster configuration: %s", err))
			return
		}
		update := kubernetes.ClusterConfiguration{
			VersionID: int(plan.VersionID.ValueInt64()),
			Variables: current.Variables,
		}

		err = retry(ctx, "update cluster configuration", func() (err error) {
			_, err = k.clusterService.UpdateConfiguration(ctx, int(state.ID.ValueInt64()), update)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to change cluster configuration: %s", err))
			return
		}
	}

	if plan.NodeCount.ValueInt64() != state.NodeCount.ValueInt64() || plan.NodeProductID.ValueInt64() != state.NodeProductID.ValueInt64() {
		if _, err := k.waitForClusterUnlocked(ctx, int(state.ID.ValueInt64())); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be unlocked: %s", err))
			return
		}

		update := kubernetes.ClusterUpdateFlavor{
			Worker: kubernetes.ClusterWorkerUpdate{
				ProductID: int(plan.NodeProductID.ValueInt64()),
				Count:     int(plan.NodeCount.ValueInt64()),
			},
		}

		err := retry(ctx, "update cluster flavor", func() (err error) {
			_, err = k.clusterService.UpdateFlavor(ctx, int(state.ID.ValueInt64()), update)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to change cluster flavor: %s", err))
			return
		}
	}

	cluster, err := k.waitForClusterUnlocked(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be unlocked: %s", err))
		return
	}

	state.FromEntity(cluster)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (k kubernetesClusterResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state kubernetesClusterResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete cluster", func() error {
		return k.clusterService.Delete(ctx, int(state.ID.ValueInt64()))
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete cluster: %s", err))
		return
	}

	if err := k.waitForClusterGone(ctx, int(state.ID.ValueInt64())); err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster deletion: %s", err))
		return
	}
}

// the create order succeeds while the cluster is still provisioning — until it
// is unlocked and healthy, updates are refused
func waitForClusterReady(ctx context.Context, service kubernetes.ClusterService, clusterID int) (cluster kubernetes.Cluster, err error) {
	err = waitFor(ctx, clusterWaitTimeout, defaultWaitInterval, fmt.Sprintf("cluster %d to be ready", clusterID), func(ctx context.Context) (bool, error) {
		got, err := service.Get(ctx, clusterID)
		if err != nil {
			return false, err
		}
		cluster = got
		return !cluster.Locked && cluster.Status.ID == compute.ClusterStatusHealthy, nil
	})

	return cluster, err
}

// configuration and flavor updates run as an async action that keeps the
// cluster locked after the call returns — any update in that window is refused
func (k kubernetesClusterResource) waitForClusterUnlocked(ctx context.Context, clusterID int) (cluster kubernetes.Cluster, err error) {
	err = waitFor(ctx, clusterWaitTimeout, defaultWaitInterval, fmt.Sprintf("cluster %d to be unlocked", clusterID), func(ctx context.Context) (bool, error) {
		got, err := k.clusterService.Get(ctx, clusterID)
		if err != nil {
			return false, err
		}
		cluster = got
		return !cluster.Locked, nil
	})

	return cluster, err
}

// cluster deletion is queued — the delete call returns while the cluster still
// exists, and deleting the network is refused until it is gone
func (k kubernetesClusterResource) waitForClusterGone(ctx context.Context, clusterID int) error {
	return waitFor(ctx, clusterWaitTimeout, defaultWaitInterval, fmt.Sprintf("cluster %d to be gone", clusterID), func(ctx context.Context) (bool, error) {
		_, err := k.clusterService.Get(ctx, clusterID)
		if statusCode(err) == http.StatusNotFound {
			return true, nil
		}
		return false, err
	})
}

func (k kubernetesClusterResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}
