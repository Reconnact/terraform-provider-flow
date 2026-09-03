package flow

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeServer_Basic(t *testing.T) {
	serverName := acctest.RandomWithPrefix("test-server")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServerConfig(t, serverName, "10.101.0.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_server.foobar", "id"),
					resource.TestCheckResourceAttr("flow_compute_server.foobar", "name", serverName),
					resource.TestCheckResourceAttr("flow_compute_server.foobar", "location_id", "1"),
					resource.TestCheckResourceAttrPair("flow_compute_server.foobar", "image_id", "data.flow_compute_image.ubuntu", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_server.foobar", "product_id", "data.flow_product.small", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_server.foobar", "network_id", "flow_compute_network.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_server.foobar", "key_pair_id", "flow_compute_key_pair.foobar", "id"),
					resource.TestCheckResourceAttrSet("flow_compute_server.foobar", "private_ip"),
					resource.TestCheckResourceAttrSet("flow_compute_server.foobar", "network_interface_id"),
				),
			},
		},
	})
}
