package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var protoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"flow": providerserver.NewProtocol6WithError(New(
		WithVersion("test"),
	)),
}

// TODO: run tests parallel once the metrics bug in the api is fixed
// testAccSequential runs a test case without t.Parallel() — the api cannot
// create two billable objects for one org at the same time (metrics table,
// "Record has changed since last read"), a parallel run fails at random.
// Back to resource.ParallelTest once that is fixed.
func testAccSequential(t *testing.T, testCase resource.TestCase) {
	t.Helper()
	resource.Test(t, testCase)
}

// testAccServerConfig is the smallest server possible:
// ubuntu on the smallest product in its own network, with a random key pair the. Tests append the resource they are about.
// The cidr must differ per test — they run in parallel.
func testAccServerConfig(t *testing.T, name string, cidr string) string {
	t.Helper()

	publicKey, _, err := acctest.RandSSHKeyPair(name)
	if err != nil {
		t.Fatal(err)
	}

	return fmt.Sprintf(`
data "flow_compute_image" "ubuntu" {
	key = "linux-ubuntu-22.04-lts"
}

data "flow_product" "small" {
	name = "b1.1x1"
}

resource "flow_compute_key_pair" "foobar" {
	name       = "%[1]s"
	public_key = "%[3]s"
}

resource "flow_compute_network" "foobar" {
	name        = "%[1]s"
	location_id = 1
	cidr        = "%[2]s"
}

resource "flow_compute_server" "foobar" {
	name        = "%[1]s"
	location_id = 1
	image_id    = data.flow_compute_image.ubuntu.id
	product_id  = data.flow_product.small.id
	network_id  = flow_compute_network.foobar.id
	key_pair_id = flow_compute_key_pair.foobar.id
}
`, name, cidr, publicKey)
}
