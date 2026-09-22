package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMacBareMetalSecurityGroup_Basic(t *testing.T) {
	t.Skip("the api allows one mac bare metal network per organisation, and dev's backend refuses new ones")

	securityGroupName := acctest.RandomWithPrefix("test-security-group")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccMacBareMetalSecurityGroupConfigBasic, securityGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_security_group.foobar", "id"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group.foobar", "name", securityGroupName),
					resource.TestCheckResourceAttrPair("flow_mac_bare_metal_security_group.foobar", "network_id", "flow_mac_bare_metal_network.foobar", "id"),
				),
			},
		},
	})
}

const testAccMacBareMetalSecurityGroupConfigBasic = `
data "flow_location" "zrh1" {
	name = "ZRH1"
}

resource "flow_mac_bare_metal_network" "foobar" {
	name        = "%[1]s"
	location_id = data.flow_location.zrh1.id
}

resource "flow_mac_bare_metal_security_group" "foobar" {
	name       = "%[1]s"
	network_id = flow_mac_bare_metal_network.foobar.id
}
`
