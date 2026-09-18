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
					resource.TestCheckResourceAttr("flow_compute_load_balancer.foobar", "public", "false"),
					resource.TestCheckNoResourceAttr("flow_compute_load_balancer.foobar", "public_ip"),
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

func TestAccComputeLoadBalancer_Public(t *testing.T) {
	name := acctest.RandomWithPrefix("test-load-balancer-public")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeLoadBalancerConfigPublic, name, "10.109.0.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_load_balancer.foobar", "id"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer.foobar", "public", "true"),
					resource.TestCheckResourceAttrSet("flow_compute_load_balancer.foobar", "public_ip"),
					resource.TestCheckResourceAttrSet("flow_compute_load_balancer.foobar", "private_ip"),
				),
			},
			{
				ResourceName:      "flow_compute_load_balancer.foobar",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccComputeLoadBalancerConfigPublic = `
resource "flow_compute_network" "foobar" {
	name        = "%[1]s"
	location_id = 1
	cidr        = "%[2]s"
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

resource "flow_compute_load_balancer" "foobar" {
	name        = "%[1]s"
	location_id = 1
	network_id  = flow_compute_network.foobar.id
	public      = true

	depends_on = [flow_compute_router_interface.foobar]
}
`
