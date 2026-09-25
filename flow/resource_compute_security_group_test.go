package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccComputeSecurityGroup_Basic(t *testing.T) {
	securityGroupName := acctest.RandomWithPrefix("test-security-group")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeSecurityGroupConfigBasic, securityGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_security_group.foobar", "id"),
					resource.TestCheckResourceAttr("flow_compute_security_group.foobar", "name", securityGroupName),
					resource.TestCheckResourceAttr("flow_compute_security_group.foobar", "location_id", "1"),
				),
			},
			{
				Config: fmt.Sprintf(testAccComputeSecurityGroupConfigBasic, securityGroupName+"-renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_compute_security_group.foobar", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_compute_security_group.foobar", "name", securityGroupName+"-renamed"),
				),
			},
			{
				ResourceName:      "flow_compute_security_group.foobar",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccComputeSecurityGroupConfigBasic = `
resource "flow_compute_security_group" "foobar" {
	name        = "%s"
	location_id = 1
}
`
