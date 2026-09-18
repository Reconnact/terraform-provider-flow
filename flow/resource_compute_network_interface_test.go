package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeNetworkInterface_Basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-network-interface")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServerConfig(t, name, "10.102.0.0/24") + fmt.Sprintf(testAccComputeNetworkInterfaceConfigBasic, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_network_interface.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_network_interface.foobar", "server_id", "flow_compute_server.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_network_interface.foobar", "network_id", "flow_compute_network.back", "id"),
					resource.TestCheckResourceAttrSet("flow_compute_network_interface.foobar", "private_ip"),
					resource.TestCheckResourceAttrSet("flow_compute_network_interface.foobar", "mac_address"),
					resource.TestCheckResourceAttrSet("flow_compute_network_interface.foobar", "security"),
				),
			},
		},
	})
}

// a second network, so the interface under test is not the primary one the server brings along
const testAccComputeNetworkInterfaceConfigBasic = `
resource "flow_compute_network" "back" {
	name        = "%s-back"
	location_id = 1
	cidr        = "10.102.1.0/24"
}

resource "flow_compute_network_interface" "foobar" {
	server_id  = flow_compute_server.foobar.id
	network_id = flow_compute_network.back.id
}
`
