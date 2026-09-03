package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeLoadBalancerPool_Basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-load-balancer-pool")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeLoadBalancerConfigBasic, name, "10.106.0.0/24") + testAccComputeLoadBalancerPoolConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_load_balancer_pool.foobar", "id"),
					resource.TestCheckResourceAttrSet("flow_compute_load_balancer_pool.foobar", "name"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_pool.foobar", "load_balancer_id", "flow_compute_load_balancer.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_pool.foobar", "entry_protocol_id", "data.flow_compute_load_balancer_protocol.http", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_pool.foobar", "target_protocol_id", "data.flow_compute_load_balancer_protocol.http", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_pool.foobar", "balancing_algorithm_id", "data.flow_compute_load_balancer_algorithm.round_robin", "id"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "entry_port", "80"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "sticky_session", "true"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_pool.foobar", "health_check.type_id", "data.flow_compute_load_balancer_health_check_type.http", "id"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.http.method", "GET"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.http.path", "/"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.interval", "10s"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.timeout", "5s"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.healthy_threshold", "2"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.unhealthy_threshold", "2"),
				),
			},
		},
	})
}

// timeout 5s is the api minimum, interval is stored in seconds and read back as a duration string
const testAccComputeLoadBalancerPoolConfigBasic = `
data "flow_compute_load_balancer_algorithm" "round_robin" {
	key = "round_robin"
}

data "flow_compute_load_balancer_protocol" "http" {
	key = "http"
}

data "flow_compute_load_balancer_health_check_type" "http" {
	key = "http"
}

resource "flow_compute_load_balancer_pool" "foobar" {
	load_balancer_id       = flow_compute_load_balancer.foobar.id
	entry_protocol_id      = data.flow_compute_load_balancer_protocol.http.id
	entry_port             = 80
	target_protocol_id     = data.flow_compute_load_balancer_protocol.http.id
	balancing_algorithm_id = data.flow_compute_load_balancer_algorithm.round_robin.id
	sticky_session         = true

	health_check = {
		type_id             = data.flow_compute_load_balancer_health_check_type.http.id
		http                = { method = "GET", path = "/" }
		interval            = "10s"
		timeout             = "5s"
		healthy_threshold   = 2
		unhealthy_threshold = 2
	}
}
`
