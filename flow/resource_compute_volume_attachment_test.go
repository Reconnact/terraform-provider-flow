package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeVolumeAttachment_Basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-volume-attachment")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServerConfig(t, name, "10.103.0.0/24") + fmt.Sprintf(testAccComputeVolumeAttachmentConfigBasic, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("flow_compute_volume_attachment.foobar", "volume_id", "flow_compute_volume.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_volume_attachment.foobar", "server_id", "flow_compute_server.foobar", "id"),
				),
			},
		},
	})
}

const testAccComputeVolumeAttachmentConfigBasic = `
resource "flow_compute_volume" "foobar" {
	name        = "%s"
	location_id = 1
	size        = 1
}

resource "flow_compute_volume_attachment" "foobar" {
	volume_id = flow_compute_volume.foobar.id
	server_id = flow_compute_server.foobar.id
}
`
