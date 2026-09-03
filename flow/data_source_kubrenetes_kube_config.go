package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/compute"
	"github.com/flowswiss/goclient/kubernetes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*kubernetesKubeConfigDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*kubernetesKubeConfigDataSource)(nil)
)

type kubernetesKubeConfigDataSourceData struct {
	ClusterID  types.Int64  `tfsdk:"cluster_id"`
	KubeConfig types.String `tfsdk:"kube_config"`
}

func (k *kubernetesKubeConfigDataSourceData) FromEntity(clusterID int, kubeConfig kubernetes.ClusterKubeConfig) {
	k.ClusterID = types.Int64Value(int64(clusterID))
	k.KubeConfig = types.StringValue(kubeConfig.KubeConfig)
}

func (k kubernetesKubeConfigDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the cluster",
				Required:            true,
			},
			"kube_config": schema.StringAttribute{
				MarkdownDescription: "kube config of the cluster",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func newKubernetesKubeConfigDataSource() datasource.DataSource {
	return &kubernetesKubeConfigDataSource{}
}

func (k *kubernetesKubeConfigDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kubernetes_kube_config"
}

func (k *kubernetesKubeConfigDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	k.clusterService = kubernetes.NewClusterService(client)
}

type kubernetesKubeConfigDataSource struct {
	clusterService kubernetes.ClusterService
}

func (k kubernetesKubeConfigDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config kubernetesKubeConfigDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	clusterID := int(config.ClusterID.ValueInt64())

	// the kube-config is refused  until the cluster is healthy and unlocked — the plain get first so a wrong id fails at once
	cluster, err := k.clusterService.Get(ctx, clusterID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get cluster: %s", err))
		return
	}
	if cluster.Locked || cluster.Status.ID != compute.ClusterStatusHealthy {
		if _, err := waitForClusterReady(ctx, k.clusterService, clusterID); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be ready: %s", err))
			return
		}
	}

	kubeConfig, err := k.clusterService.GetKubeConfig(ctx, clusterID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get kube config: %s", err))
		return
	}

	var state kubernetesKubeConfigDataSourceData
	state.FromEntity(int(config.ClusterID.ValueInt64()), kubeConfig)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
