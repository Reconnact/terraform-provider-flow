package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeRouter_Basic(t *testing.T) {
	routerName := acctest.RandomWithPrefix("test-router")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeRouterConfigBasic, "foobar_public", routerName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_router.foobar_public", "id"),
					resource.TestCheckResourceAttr("flow_compute_router.foobar_public", "name", routerName),
					resource.TestCheckResourceAttr("flow_compute_router.foobar_public", "location_id", "1"),
					resource.TestCheckResourceAttr("flow_compute_router.foobar_public", "public", "true"),
					resource.TestCheckResourceAttrSet("flow_compute_router.foobar_public", "public_ip"),
				),
			},
			{
				Config: fmt.Sprintf(testAccComputeRouterConfigBasic, "foobar_private", routerName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_router.foobar_private", "id"),
					resource.TestCheckResourceAttr("flow_compute_router.foobar_private", "name", routerName),
					resource.TestCheckResourceAttr("flow_compute_router.foobar_private", "location_id", "1"),
					resource.TestCheckResourceAttr("flow_compute_router.foobar_private", "public", "false"),
					resource.TestCheckNoResourceAttr("flow_compute_router.foobar_private", "public_ip"),
				),
			},
		},
	})
}

const testAccComputeRouterConfigBasic = `
resource "flow_compute_router" "%s" {
	name        = "%s"
	location_id = 1

	public = %t
}
`
