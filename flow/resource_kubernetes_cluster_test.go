package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKubernetesCluster_Basic(t *testing.T) {
	//TODO: Investigate dev issue
	t.Skip("dev never runs the cluster-delete job")

	clusterName := acctest.RandomWithPrefix("test-cluster")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccKubernetesClusterConfigBasic, clusterName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "id"),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "name", clusterName),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "location_id", "1"),
					resource.TestCheckResourceAttrPair("flow_kubernetes_cluster.foobar", "network_id", "flow_compute_network.foobar", "id"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "security_group_id"),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "public", "true"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "public_address"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "dns_name"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "version_id"),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "node_count", "3"),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "node_product_id", "44"),
				),
			},
		},
	})
}

// a public cluster only comes up in a network behind a public router — the
// router and its interface are the fixture, the cluster is what is tested
const testAccKubernetesClusterConfigBasic = `
resource "flow_compute_network" "foobar" {
	name        = "%[1]s"
	location_id = 1
	cidr        = "10.108.0.0/24"
}

resource "flow_compute_router" "foobar" {
	name        = "%[1]s"
	location_id = 1
	public      = true
}

resource "flow_compute_router_interface" "foobar" {
	router_id  = flow_compute_router.foobar.id
	network_id = flow_compute_network.foobar.id
}

resource "flow_kubernetes_cluster" "foobar" {
	name = "%[1]s"

	location_id = 1
	network_id  = flow_compute_network.foobar.id

	public = true

	node_count      = 3
	node_product_id = 44

	depends_on = [flow_compute_router_interface.foobar]
}
`
