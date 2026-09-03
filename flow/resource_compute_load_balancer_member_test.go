package flow

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeLoadBalancerMember_Basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-load-balancer-member")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServerConfig(t, name, "10.107.0.0/24") + testAccComputeLoadBalancerMemberConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_load_balancer_member.foobar", "id"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_member.foobar", "name", "server-1"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_member.foobar", "load_balancer_id", "flow_compute_load_balancer.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_member.foobar", "pool_id", "flow_compute_load_balancer_pool.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_member.foobar", "address", "flow_compute_server.foobar", "private_ip"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_member.foobar", "port", "8080"),
				),
			},
		},
	})
}

// a member has to be an address in the load balancer's network, so it needs a server there
const testAccComputeLoadBalancerMemberConfigBasic = `
resource "flow_compute_load_balancer" "foobar" {
	name        = flow_compute_server.foobar.name
	location_id = 1
	network_id  = flow_compute_network.foobar.id
}
` + testAccComputeLoadBalancerPoolConfigBasic + `
resource "flow_compute_load_balancer_member" "foobar" {
	load_balancer_id = flow_compute_load_balancer.foobar.id
	pool_id          = flow_compute_load_balancer_pool.foobar.id
	name             = "server-1"
	address          = flow_compute_server.foobar.private_ip
	port             = 8080
}
`
