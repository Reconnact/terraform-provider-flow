package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeLoadBalancer_Basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-load-balancer")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeLoadBalancerConfigBasic, name, "10.105.0.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_load_balancer.foobar", "id"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer.foobar", "name", name),
					resource.TestCheckResourceAttr("flow_compute_load_balancer.foobar", "location_id", "1"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer.foobar", "network_id", "flow_compute_network.foobar", "id"),
					resource.TestCheckResourceAttrSet("flow_compute_load_balancer.foobar", "private_ip"),
				),
			},
		},
	})
}

const testAccComputeLoadBalancerConfigBasic = `
resource "flow_compute_network" "foobar" {
	name        = "%[1]s"
	location_id = 1
	cidr        = "%[2]s"
}

resource "flow_compute_load_balancer" "foobar" {
	name        = "%[1]s"
	location_id = 1
	network_id  = flow_compute_network.foobar.id
}
`
