package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccMacBareMetalSecurityGroupRule_Basic(t *testing.T) {
	securityGroupName := acctest.RandomWithPrefix("test-security-group")

	protocolNumber := "6"
	protocolName := "tcp"
	fromPort := 22
	toPort := 22
	ipRange := "1.1.1.1/32"

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccMacBareMetalSecurityGroupRuleConfigBasic, securityGroupName, protocolName, fromPort, toPort, ipRange),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_security_group_rule.foobar", "id"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "direction", "ingress"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "protocol.number", protocolNumber),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "protocol.name", protocolName),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "port_range.from", fmt.Sprint(fromPort)),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "port_range.to", fmt.Sprint(toPort)),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "ip_range", ipRange),
					resource.TestCheckNoResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "icmp"),
				),
			},
			{
				// mac bare metal has no check on the pair, so a dropped code 0 reaches the
				// vendor api as null and the rule comes back wider than asked for: the values
				// read back are the assertion, not the absence of an error
				Config: fmt.Sprintf(testAccMacBareMetalSecurityGroupRuleConfigICMP, securityGroupName, 8, 0, ipRange),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_mac_bare_metal_security_group_rule.foobar", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "protocol.number", "1"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "protocol.name", "icmp"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "icmp.type", "8"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "icmp.code", "0"),
					resource.TestCheckNoResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "port_range"),
				),
			},
		},
	})
}

// one mac bare metal network per org — see the security group test
const testAccMacBareMetalSecurityGroupRuleConfigBasic = `
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

resource "flow_mac_bare_metal_security_group_rule" "foobar" {
	security_group_id = flow_mac_bare_metal_security_group.foobar.id

	direction = "ingress"
	protocol  = { name = "%[2]s" }

	port_range = {
		from = %[3]d
		to   = %[4]d
	}

	ip_range = "%[5]s"
}
`

const testAccMacBareMetalSecurityGroupRuleConfigICMP = `
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

resource "flow_mac_bare_metal_security_group_rule" "foobar" {
	security_group_id = flow_mac_bare_metal_security_group.foobar.id

	direction = "ingress"
	protocol  = { name = "icmp" }

	icmp = {
		type = %[2]d
		code = %[3]d
	}

	ip_range = "%[4]s"
}
`
