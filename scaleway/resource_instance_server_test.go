package scaleway_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/acctest"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/httperrors"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/iam"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/provider"
	"github.com/stretchr/testify/require"
)

func init() {
	resource.AddTestSweepers("scaleway_instance_server", &resource.Sweeper{
		Name: "scaleway_instance_server",
		F:    testSweepInstanceServer,
	})
}

func testSweepInstanceServer(_ string) error {
	return sweepZones(scw.AllZones, func(scwClient *scw.Client, zone scw.Zone) error {
		instanceAPI := instance.NewAPI(scwClient)
		logging.L.Debugf("sweeper: destroying the instance server in (%s)", zone)
		listServers, err := instanceAPI.ListServers(&instance.ListServersRequest{Zone: zone}, scw.WithAllPages())
		if err != nil {
			logging.L.Warningf("error listing servers in (%s) in sweeper: %s", zone, err)
			return nil
		}

		for _, srv := range listServers.Servers {
			if srv.State == instance.ServerStateStopped || srv.State == instance.ServerStateStoppedInPlace {
				err := instanceAPI.DeleteServer(&instance.DeleteServerRequest{
					Zone:     zone,
					ServerID: srv.ID,
				})
				if err != nil {
					return fmt.Errorf("error deleting server in sweeper: %s", err)
				}
			} else if srv.State == instance.ServerStateRunning {
				_, err := instanceAPI.ServerAction(&instance.ServerActionRequest{
					Zone:     zone,
					ServerID: srv.ID,
					Action:   instance.ServerActionTerminate,
				})
				if err != nil {
					return fmt.Errorf("error terminating server in sweeper: %s", err)
				}
			}
		}

		return nil
	})
}

func TestAccScalewayInstanceServer_Minimal1(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_server" "base" {
					  image = "ubuntu_focal"
					  type  = "DEV1-S"
					
					  tags = [ "terraform-test", "scaleway_instance_server", "minimal" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "image", "ubuntu_focal"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "type", "DEV1-S"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "root_volume.0.delete_on_termination", "true"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "root_volume.0.size_in_gb", "20"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "root_volume.0.volume_id"),
					testAccCheckScalewayInstanceServerHasNewVolume(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "enable_dynamic_ip", "false"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.0", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.1", "scaleway_instance_server"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.2", "minimal"),
				),
			},
			{
				// Image label such as ubuntu_focal
				Config: `
					resource "scaleway_instance_server" "base" {
					  image = "ubuntu_focal"
					  type  = "DEV1-S"
					
					  tags = [ "terraform-test", "scaleway_instance_server", "minimal" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "image", "ubuntu_focal"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "type", "DEV1-S"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "root_volume.0.delete_on_termination", "true"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "root_volume.0.size_in_gb", "20"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "root_volume.0.volume_id"),
					testAccCheckScalewayInstanceServerHasNewVolume(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.0", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.1", "scaleway_instance_server"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.2", "minimal"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_Minimal2(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_server" "base" {
					  image = "ubuntu_focal"
					  type  = "DEV1-S"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "image", "ubuntu_focal"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "type", "DEV1-S"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "root_volume.0.delete_on_termination", "true"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "root_volume.0.size_in_gb", "20"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "root_volume.0.volume_id"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "enable_dynamic_ip", "false"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "state", "started"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_server" main {
					  image = "ubuntu_focal"
					  type  = "DEV1-S"
					  root_volume {
						volume_type = "l_ssd"
					  }
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "image", "ubuntu_focal"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "type", "DEV1-S"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "root_volume.0.delete_on_termination", "true"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "root_volume.0.size_in_gb", "20"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "root_volume.0.volume_type", "l_ssd"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.main", "root_volume.0.volume_id"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "enable_dynamic_ip", "false"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_server" main2 {
					  image = "ubuntu_focal"
					  type  = "DEV1-S"
					  root_volume {
						volume_type = "l_ssd"
						size_in_gb  = 20
					  }
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.main2"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main2", "image", "ubuntu_focal"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main2", "type", "DEV1-S"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main2", "root_volume.0.delete_on_termination", "true"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main2", "root_volume.0.size_in_gb", "20"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main2", "root_volume.0.volume_type", "l_ssd"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.main2", "root_volume.0.volume_id"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main2", "enable_dynamic_ip", "false"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_RootVolume1(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						root_volume {
							size_in_gb = 10
							delete_on_termination = true
						}
						tags = [ "terraform-test", "scaleway_instance_server", "root_volume" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					testAccCheckScalewayInstanceServerHasNewVolume(tt, "scaleway_instance_server.base"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_RootVolume_Boot(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						state = "stopped"
						root_volume {
							boot = true
							delete_on_termination = true
						}
						tags = [ "terraform-test", "scaleway_instance_server", "root_volume" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "root_volume.0.boot", "true"),
					testAccCheckScalewayInstanceServerHasNewVolume(tt, "scaleway_instance_server.base"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						state = "stopped"
						root_volume {
							boot = false
							delete_on_termination = true
						}
						tags = [ "terraform-test", "scaleway_instance_server", "root_volume" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "root_volume.0.boot", "false"),
					testAccCheckScalewayInstanceServerHasNewVolume(tt, "scaleway_instance_server.base"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_RootVolume_ID(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "server_volume" {
					  type       = "b_ssd"
					  name       = "tf_tests_rootvolume"
					  size_in_gb = 10
					}

					resource "scaleway_instance_server" "base" {
						type  = "DEV1-S"
						state = "stopped"
						root_volume {
							volume_id = scaleway_instance_volume.server_volume.id
							boot = true
							delete_on_termination = false
						}
						tags = [ "terraform-test", "scaleway_instance_server", "root_volume" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.base", "root_volume.0.volume_id", "scaleway_instance_volume.server_volume", "id"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.base", "root_volume.0.size_in_gb", "scaleway_instance_volume.server_volume", "size_in_gb"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_Basic(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				// DEV1-M
				Config: `
					data "scaleway_marketplace_image" "ubuntu" {
					  instance_type   = "DEV1-M"
					  label         = "ubuntu_focal"
					}
					
					resource "scaleway_instance_server" "base" {
					  name  = "test"
					  image = "${data.scaleway_marketplace_image.ubuntu.id}"
					  type  = "DEV1-M"
					
					  tags = [ "terraform-test", "scaleway_instance_server", "basic" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "type", "DEV1-M"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "name", "test"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.0", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.1", "scaleway_instance_server"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.2", "basic"),
				),
			},
			{
				// DEV1-S
				Config: `
					data "scaleway_marketplace_image" "ubuntu" {
					  instance_type   = "DEV1-S"
					  label         = "ubuntu_focal"
					}
					
					resource "scaleway_instance_server" "base" {
					  name  = "test"
					  image = "${data.scaleway_marketplace_image.ubuntu.id}"
					  type  = "DEV1-S"
					  replace_on_type_change  = true
					
					  tags = [ "terraform-test", "scaleway_instance_server", "basic" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "type", "DEV1-S"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "name", "test"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.0", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.1", "scaleway_instance_server"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.2", "basic"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_State1(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				// started
				Config: `
					data "scaleway_marketplace_image" "ubuntu" {
					  instance_type = "DEV1-S"
					  label         = "ubuntu_focal"
					}
					
					resource "scaleway_instance_server" "base" {
					  image = "${data.scaleway_marketplace_image.ubuntu.id}"
					  type  = "DEV1-S"
					  state = "started"
					  tags  = [ "terraform-test", "scaleway_instance_server", "state" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "state", "started"),
				),
			},
			{
				// standby
				Config: `
					data "scaleway_marketplace_image" "ubuntu" {
					  instance_type = "DEV1-S"
					  label         = "ubuntu_focal"
					}
					
					resource "scaleway_instance_server" "base" {
					  image = "${data.scaleway_marketplace_image.ubuntu.id}"
					  type  = "DEV1-S"
					  state = "standby"
					  tags  = [ "terraform-test", "scaleway_instance_server", "state" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "state", "standby"),
				),
			},
			{
				// stopped
				Config: `
					data "scaleway_marketplace_image" "ubuntu" {
					  instance_type = "DEV1-S"
					  label         = "ubuntu_focal"
					}
					
					resource "scaleway_instance_server" "base" {
					  image = "${data.scaleway_marketplace_image.ubuntu.id}"
					  type  = "DEV1-S"
					  state = "stopped"
					  tags  = [ "terraform-test", "scaleway_instance_server", "state" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "state", "stopped"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_State2(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				// stopped
				Config: `
					data "scaleway_marketplace_image" "ubuntu" {
					  instance_type = "DEV1-S"
					  label         = "ubuntu_focal"
					}
					
					resource "scaleway_instance_server" "base" {
					  image = "${data.scaleway_marketplace_image.ubuntu.id}"
					  type  = "DEV1-S"
					  state = "stopped"
					  tags  = [ "terraform-test", "scaleway_instance_server", "state" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "state", "stopped"),
				),
			},
			{
				// standby
				Config: `
					data "scaleway_marketplace_image" "ubuntu" {
					  instance_type = "DEV1-S"
					  label         = "ubuntu_focal"
					}
					
					resource "scaleway_instance_server" "base" {
					  image = "${data.scaleway_marketplace_image.ubuntu.id}"
					  type  = "DEV1-S"
					  state = "standby"
					  tags  = [ "terraform-test", "scaleway_instance_server", "state" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "state", "standby"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_UserData_WithCloudInitAtStart(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
				resource "scaleway_instance_server" "base" {
					image = "ubuntu_focal"
					type  = "DEV1-S"

					user_data = {
				   		foo   = "bar"
						cloud-init =  <<EOF
#cloud-config
apt_update: true
apt_upgrade: true
EOF 
				 	}
				}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "user_data.foo", "bar"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "user_data.cloud-init", "#cloud-config\napt_update: true\napt_upgrade: true\n"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_UserData_WithoutCloudInitAtStart(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				// Without cloud-init
				Config: `
					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						root_volume {
							size_in_gb = 20
						}
						tags  = [ "terraform-test", "scaleway_instance_server", "user_data" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "user_data.%", "0"),
				),
			},
			{
				// With cloud-init
				Config: `
					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						tags  = [ "terraform-test", "scaleway_instance_server", "user_data" ]
						root_volume {
							size_in_gb = 20
						}
						user_data = {
							cloud-init = <<EOF
#cloud-config
apt_update: true
apt_upgrade: true
EOF
						}
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "user_data.cloud-init", "#cloud-config\napt_update: true\napt_upgrade: true\n"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_AdditionalVolumes(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				// With additional local
				Config: `
					resource "scaleway_instance_volume" "local" {
						size_in_gb = 10
						type = "l_ssd"
					}

					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type = "DEV1-S"
						
						root_volume {
							size_in_gb = 10
						}

						tags = [ "terraform-test", "scaleway_instance_server", "additional_volume_ids" ]

						additional_volume_ids = [
							scaleway_instance_volume.local.id
						]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "root_volume.0.size_in_gb", "10"),
				),
			},
			{
				// With additional local and block
				Config: `
					resource "scaleway_instance_volume" "local" {
						size_in_gb = 10
						type = "l_ssd"
					}

					resource "scaleway_instance_volume" "block" {
						size_in_gb = 10
						type = "b_ssd"
					}

					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type = "DEV1-S"
						
						root_volume {
							size_in_gb = 10
						}

						tags = [ "terraform-test", "scaleway_instance_server", "additional_volume_ids" ]

						additional_volume_ids = [
							scaleway_instance_volume.local.id,
							scaleway_instance_volume.block.id
						]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceVolumeExists(tt, "scaleway_instance_volume.block"),
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.block", "size_in_gb", "10"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "root_volume.0.size_in_gb", "10"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_AdditionalVolumesDetach(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayInstanceVolumeDestroy(tt),
			testAccCheckScalewayInstanceServerDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: `
					variable "zone" {
						type    = string
						default = "fr-par-1"
					}

					resource "scaleway_instance_volume" "main" {
  						type       = "b_ssd"
  						name       = "foobar"
  						size_in_gb = 1
					}

					resource "scaleway_instance_server" "main" {
						type  = "DEV1-S"
  						image = "ubuntu_focal"
  						name  = "foobar"

						enable_dynamic_ip = true

						additional_volume_ids = [scaleway_instance_volume.main.id]
					}
				`,
			},
			{
				Config: `
					variable "zone" {
						type    = string
						default = "fr-par-1"
					}

					resource "scaleway_instance_volume" "main" {
  						type       = "b_ssd"
  						name       = "foobar"
  						size_in_gb = 1
					}

					resource "scaleway_instance_server" "main" {
						type  = "DEV1-S"
  						image = "ubuntu_focal"
  						name  = "foobar"

						enable_dynamic_ip = true

						additional_volume_ids = []
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceVolumeExists(tt, "scaleway_instance_volume.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "additional_volume_ids.#", "0"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_WithPlacementGroup(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_placement_group" "ha" {
						policy_mode = "enforced"
						policy_type = "max_availability"
					}
					
					resource "scaleway_instance_server" "base" {
						count = 3
						image = "ubuntu_focal"
						type  = "DEV1-S"
						placement_group_id = "${scaleway_instance_placement_group.ha.id}"
						tags  = [ "terraform-test", "scaleway_instance_server", "placement_group" ]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base.0"),
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base.1"),
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base.2"),
					testAccCheckScalewayInstancePlacementGroupExists(tt, "scaleway_instance_placement_group.ha"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base.0", "placement_group_policy_respected", "true"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base.1", "placement_group_policy_respected", "true"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base.2", "placement_group_policy_respected", "true"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_Ipv6(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_server" "server01" {
						image = "ubuntu_focal"
		  				type  = "DEV1-S"
		  				enable_ipv6 = true
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.server01"),
					testCheckResourceAttrIPv6("scaleway_instance_server.server01", "ipv6_address"),
					testCheckResourceAttrIPv6("scaleway_instance_server.server01", "ipv6_gateway"),
					resource.TestCheckResourceAttr("scaleway_instance_server.server01", "ipv6_prefix_length", "64"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_server" "server01" {
						image = "ubuntu_focal"
		  				type  = "DEV1-S"
		  				enable_ipv6 = false
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.server01"),
					resource.TestCheckResourceAttr("scaleway_instance_server.server01", "ipv6_address", ""),
					resource.TestCheckResourceAttr("scaleway_instance_server.server01", "ipv6_gateway", ""),
					resource.TestCheckResourceAttr("scaleway_instance_server.server01", "ipv6_prefix_length", "0"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_Basic2(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					data "scaleway_marketplace_image" "ubuntu" {
					  instance_type   = "DEV1-M"
					  label         = "ubuntu_focal"
					}

					resource "scaleway_instance_server" "server01" {
						type  = "DEV1-S"
						image = data.scaleway_marketplace_image.ubuntu.id
						state = "stopped"
					}
				`,
			},
			{
				ResourceName: "scaleway_instance_server.server01",
				ImportState:  true,
			},
		},
	})
}

func TestAccScalewayInstanceServer_WithReservedIP(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_ip" "first" {}
					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						ip_id = scaleway_instance_ip.first.id
						tags  = [ "terraform-test", "scaleway_instance_server", "reserved_ip" ]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttrPair("scaleway_instance_ip.first", "address", "scaleway_instance_server.base", "public_ip"),
					resource.TestCheckResourceAttrPair("scaleway_instance_ip.first", "id", "scaleway_instance_server.base", "ip_id"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_ip" "first" {}
					resource "scaleway_instance_ip" "second" {}
					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						ip_id = scaleway_instance_ip.second.id
						tags  = [ "terraform-test", "scaleway_instance_server", "reserved_ip" ]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					testAccCheckScalewayInstanceIPPairWithServer(tt, "scaleway_instance_ip.second", "scaleway_instance_server.base"),
					resource.TestCheckResourceAttrPair("scaleway_instance_ip.second", "address", "scaleway_instance_server.base", "public_ip"),
					resource.TestCheckResourceAttrPair("scaleway_instance_ip.second", "id", "scaleway_instance_server.base", "ip_id"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_ip" "first" {}
					resource "scaleway_instance_ip" "second" {}
					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						tags  = [ "terraform-test", "scaleway_instance_server", "reserved_ip" ]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					testAccCheckScalewayInstanceServerNoIPAssigned(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "public_ip", ""),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "ip_id", ""),
				),
			},
			{
				Config: `
					resource "scaleway_instance_ip" "first" {}
					resource "scaleway_instance_ip" "second" {}
					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						enable_dynamic_ip = true
						tags  = [ "terraform-test", "scaleway_instance_server", "reserved_ip" ]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					testAccCheckScalewayInstanceServerNoIPAssigned(tt, "scaleway_instance_server.base"),
					testCheckResourceAttrIPv4("scaleway_instance_server.base", "public_ip"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "ip_id", ""),
				),
			},
		},
	})
}

func testAccCheckScalewayInstanceServerExists(tt *acctest.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		instanceAPI, zone, ID, err := scaleway.InstanceAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = instanceAPI.GetServer(&instance.GetServerRequest{ServerID: ID, Zone: zone})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayInstancePrivateNICsExists(tt *acctest.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		instanceAPI, zone, ID, err := scaleway.InstanceAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		res, err := instanceAPI.ListPrivateNICs(&instance.ListPrivateNICsRequest{ServerID: ID, Zone: zone})
		if err != nil {
			return err
		}

		privateNetworksOnServer := make(map[string]struct{})
		// build current private networks on server
		for _, key := range res.PrivateNics {
			privateNetworksOnServer[key.PrivateNetworkID] = struct{}{}
		}

		privateNetworksToCheckOnSchema := make(map[string]struct{})
		// build terraform private networks
		for key, value := range rs.Primary.Attributes {
			if strings.Contains(key, "pn_id") {
				privateNetworksToCheckOnSchema[locality.ExpandID(value)] = struct{}{}
			}
		}

		// check if private networks are present on server
		for pnKey := range privateNetworksToCheckOnSchema {
			if _, exist := privateNetworksOnServer[pnKey]; !exist {
				return errors.New("private network does not exist")
			}
		}

		return nil
	}
}

func testAccCheckScalewayInstanceServerDestroy(tt *acctest.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_instance_server" {
				continue
			}

			instanceAPI, zone, ID, err := scaleway.InstanceAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = instanceAPI.GetServer(&instance.GetServerRequest{
				ServerID: ID,
				Zone:     zone,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("server (%s) still exists", rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !httperrors.Is404(err) {
				return err
			}
		}

		return nil
	}
}

// testAccCheckScalewayInstanceServerHasNewVolume tests if volume name is generated by terraform
// It is useful as volume should not be set in request when creating an instance from an image
func testAccCheckScalewayInstanceServerHasNewVolume(_ *acctest.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		rootVolumeName, ok := rs.Primary.Attributes["root_volume.0.name"]
		if !ok {
			return errors.New("instance root_volume has no name")
		}

		if strings.HasPrefix(rootVolumeName, "tf") {
			return fmt.Errorf("root volume name is generated by provider, should be generated by api (%s)", rootVolumeName)
		}

		return nil
	}
}

func TestAccScalewayInstanceServer_Bootscript(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	// Quick tip to get all the different bootscript:
	// curl -sH "X-Auth-Token: $(scw config get secret-key)" https://api.scaleway.com/instance/v1/zones/fr-par-1/bootscripts | jq -r '.bootscripts[] | [.id, .architecture, .title] | @tsv'
	bootscript := "7decf961-d3e9-4711-93c7-b16c254e99b9"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_instance_server" "base" {
						type  = "DEV1-S"
						image = "ubuntu_focal"
						boot_type = "bootscript"
						bootscript_id = "%s"
					}
				`, bootscript),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "bootscript_id", bootscript),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_AlterTags(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_server" "base" {
						type  = "DEV1-L"
						image = "ubuntu_focal"
						state = "stopped"
						tags = [ "front", "web" ]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.0", "front"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.1", "web"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_server" "base" {
						type  = "DEV1-L"
						state = "stopped"
						image = "ubuntu_focal"
						tags = [ "front" ]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "tags.0", "front"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_WithDefaultRootVolumeAndAdditionalVolume(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "data" {
						size_in_gb = 100
						type = "b_ssd"
					}

					resource "scaleway_instance_server" "main" {
						type = "DEV1-S"
						image = "ubuntu-bionic"
						root_volume {
							delete_on_termination = false
					  	}
						additional_volume_ids = [ scaleway_instance_volume.data.id ]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.main"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_Enterprise(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_server" "main" {
						type  = "ENT1-S"
						image = "ubuntu_focal"
						zone  = "fr-par-2"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.main"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_ServerWithBlockNonDefaultZone(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "b_ssd"
						name       = "main"
						size_in_gb = 1
						zone       = "nl-ams-1"
					}

					resource "scaleway_instance_server" "main" {
						zone              = "nl-ams-1"
						image             = "ubuntu_focal"
						type              = "DEV1-S"
						root_volume {
							delete_on_termination = true
							size_in_gb            = 20
						}
						additional_volume_ids = [scaleway_instance_volume.main.id]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.main"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_PrivateNetwork(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_vpc_private_network internal {
						name = "private_network_instance"
					}

					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						zone = "fr-par-2"

						private_network {
							pn_id = scaleway_vpc_private_network.internal.id
						}
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "private_network.#", "1"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.pn_id"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.mac_address"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.status"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.zone"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.base", "private_network.0.pn_id",
						"scaleway_vpc_private_network.internal", "id"),
				),
			},
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "private_network_instance"
					}

					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						zone  = "fr-par-1"

						private_network {
							pn_id = scaleway_vpc_private_network.pn01.id
						}
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "private_network.#", "1"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.pn_id"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.mac_address"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.status"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.zone"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.base", "private_network.0.pn_id",
						"scaleway_vpc_private_network.pn01", "id"),
				),
			},
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "private_network_instance"
					}

					resource scaleway_vpc_private_network pn02 {
						name = "private_network_instance_02"
					}

					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"

						private_network {
							pn_id = scaleway_vpc_private_network.pn02.id
						}
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "private_network.#", "1"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.pn_id"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.mac_address"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.status"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.zone"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.base", "private_network.0.pn_id",
						"scaleway_vpc_private_network.pn02", "id"),
				),
			},
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "private_network_instance"
					}

					resource scaleway_vpc_private_network pn02 {
						name = "private_network_instance_02"
					}

					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"

						private_network {
							pn_id = scaleway_vpc_private_network.pn02.id
						}

						private_network {
							pn_id = scaleway_vpc_private_network.pn01.id
						}
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.base",
						"private_network.0.pn_id",
						"scaleway_vpc_private_network.pn02", "id"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "private_network.#", "2"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.pn_id"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.mac_address"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.status"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.0.zone"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.base", "private_network.1.pn_id",
						"scaleway_vpc_private_network.pn01", "id"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.1.pn_id"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.1.mac_address"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.1.status"),
					resource.TestCheckResourceAttrSet("scaleway_instance_server.base", "private_network.1.zone"),
				),
			},
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "private_network_instance"
					}

					resource scaleway_vpc_private_network pn02 {
						name = "private_network_instance_02"
					}

					resource "scaleway_instance_server" "base" {
						image = "ubuntu_focal"
						type  = "DEV1-S"	
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.base"),
					resource.TestCheckResourceAttr("scaleway_instance_server.base", "private_network.#", "0"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_Migrate(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_server" "main" {
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "type", "PRO2-XXS"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_server" "main" {
						image = "ubuntu_jammy"
						type  = "PRO2-XS"
						
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "type", "PRO2-XS"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_server" "main" {
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "type", "PRO2-XXS"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_MigrateInvalidLocalVolumeSize(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_server" "main" {
						image = "ubuntu_jammy"
						type  = "DEV1-L"
						
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "type", "DEV1-L"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_server" "main" {
						image = "ubuntu_jammy"
						type  = "DEV1-S"
						
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "type", "DEV1-S"),
				),
				ExpectError: regexp.MustCompile("cannot change server type"),
				PlanOnly:    true,
			},
		},
	})
}

func TestAccScalewayInstanceServer_CustomDiffImage(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_server" "main" {
						image = "ubuntu_jammy"
						type = "DEV1-S"
						state = "stopped"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "image", "ubuntu_jammy"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_server" "main" {
						image = "ubuntu_jammy"
						type = "DEV1-S"
						state = "stopped"
					}
					resource "scaleway_instance_server" "copy" {
						image = "ubuntu_jammy"
						type = "DEV1-S"
						state = "stopped"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.main"),
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.copy"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "image", "ubuntu_jammy"),
					resource.TestCheckResourceAttr("scaleway_instance_server.copy", "image", "ubuntu_jammy"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "id", "scaleway_instance_server.copy", "id"),
				),
				ResourceName: "scaleway_instance_server.copy",
				ImportState:  true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					return state.RootModule().Resources["scaleway_instance_server.main"].Primary.ID, nil
				},
				ImportStatePersist: true,
			},
			{
				Config: `
					data "scaleway_marketplace_image" "jammy" {
						label = "ubuntu_jammy"
					}
					resource "scaleway_instance_server" "main" {
						image = data.scaleway_marketplace_image.jammy.id
						type = "DEV1-S"
						state = "stopped"
					}
					resource "scaleway_instance_server" "copy" {
						image = "ubuntu_jammy"
						type = "DEV1-S"
						state = "stopped"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.main"),
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.copy"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "image", "data.scaleway_marketplace_image.jammy", "id"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "id", "scaleway_instance_server.copy", "id"),
				),
			},
			{
				RefreshState: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.main"),
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.copy"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "image", "data.scaleway_marketplace_image.jammy", "id"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "id", "scaleway_instance_server.copy", "id"),
				),
			},
			{
				Config: `
					data "scaleway_marketplace_image" "focal" {
						label = "ubuntu_focal"
					}
					resource "scaleway_instance_server" "main" {
						image = data.scaleway_marketplace_image.focal.id
						type = "DEV1-S"
						state = "stopped"
					}
					resource "scaleway_instance_server" "copy" {
						image = "ubuntu_jammy"
						type = "DEV1-S"
						state = "stopped"
					}
				`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "image", "data.scaleway_marketplace_image.focal", "id"),
					resource.TestCheckResourceAttr("scaleway_instance_server.copy", "image", "ubuntu_jammy"),
					testAccCheckScalewayInstanceServerIDsAreDifferent("scaleway_instance_server.main", "scaleway_instance_server.copy"),
				),
			},
		},
	})
}

func testAccCheckScalewayInstanceServerIDsAreDifferent(nameFirst, nameSecond string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[nameFirst]
		if !ok {
			return fmt.Errorf("resource was not found: %s", nameFirst)
		}
		idFirst := rs.Primary.ID

		rs, ok = s.RootModule().Resources[nameSecond]
		if !ok {
			return fmt.Errorf("resource was not found: %s", nameSecond)
		}
		idSecond := rs.Primary.ID

		if idFirst == idSecond {
			return fmt.Errorf("IDs of both resources were equal when they should not have been (%s and %s)", nameFirst, nameSecond)
		}
		return nil
	}
}

func TestAccScalewayInstanceServer_RoutedIPEnable(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_server" "main" {
						name  = "tf-tests-instance-server-routedip"
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "routed_ip_enabled", "false"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_server" "main" {
						name  = "tf-tests-instance-server-routedip"
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						routed_ip_enabled = true
						state = "stopped"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "routed_ip_enabled", "true"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_RoutedIPEnableWithIP(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_ip" "main" {
					}

					resource "scaleway_instance_server" "main" {
						name  = "tf-tests-instance-server-routedip-enable-with-ip"
						ip_id = scaleway_instance_ip.main.id
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "routed_ip_enabled", "false"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_ip" "main" {
					}

					resource "scaleway_instance_server" "main" {
						name  = "tf-tests-instance-server-routedip-enable-with-ip"
						ip_id = scaleway_instance_ip.main.id
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
						routed_ip_enabled = true
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "routed_ip_enabled", "true"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_IPs(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_ip" "ip1" {
						type = "routed_ipv4"
					}

					resource "scaleway_instance_server" "main" {
						name = "tf-tests-instance-server-ips"
						ip_ids = [scaleway_instance_ip.ip1.id]
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "routed_ip_enabled", "true"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "public_ips.#", "1"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "public_ips.0.id", "scaleway_instance_ip.ip1", "id"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_ip" "ip1" {
						type = "routed_ipv4"
					}

					resource "scaleway_instance_ip" "ip2" {
						type = "routed_ipv4"
					}

					resource "scaleway_instance_server" "main" {
						name = "tf-tests-instance-server-ips"
						ip_ids = [scaleway_instance_ip.ip1.id, scaleway_instance_ip.ip2.id]
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "routed_ip_enabled", "true"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "public_ips.#", "2"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "public_ips.0.id", "scaleway_instance_ip.ip1", "id"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "public_ips.1.id", "scaleway_instance_ip.ip2", "id"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_ip" "ip1" {
						type = "routed_ipv4"
					}

					resource "scaleway_instance_ip" "ip2" {
						type = "routed_ipv4"
					}

					resource "scaleway_instance_server" "main" {
						name = "tf-tests-instance-server-ips"
						ip_ids = [scaleway_instance_ip.ip2.id]
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "routed_ip_enabled", "true"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "public_ips.#", "1"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "public_ips.0.id", "scaleway_instance_ip.ip2", "id"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_IPRemoved(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_ip" "main" {}

					resource "scaleway_instance_server" "main" {
						name = "tf-tests-instance-server-ip-removed"
						ip_id = scaleway_instance_ip.main.id
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "public_ips.#", "1"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "public_ips.0.id", "scaleway_instance_ip.main", "id"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "public_ips.0.address", "scaleway_instance_server.main", "public_ip"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_ip" "main" {}

					resource "scaleway_instance_server" "main" {
						name = "tf-tests-instance-server-ip-removed"
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					testAccCheckScalewayInstanceServerNoIPAssigned(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "public_ips.#", "0"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_IPsRemoved(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_ip" "main" {
						type = "routed_ipv4"
					}

					resource "scaleway_instance_server" "main" {
						name = "tf-tests-instance-server-ips-removed"
						ip_ids = [scaleway_instance_ip.main.id]
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "routed_ip_enabled", "true"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "public_ips.#", "1"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "public_ips.0.id", "scaleway_instance_ip.main", "id"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_ip" "main" {
						type = "routed_ipv4"
					}

					resource "scaleway_instance_server" "main" {
						name = "tf-tests-instance-server-ips-removed"
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					testAccCheckScalewayInstanceServerNoIPAssigned(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "routed_ip_enabled", "true"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "public_ips.#", "0"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceServer_IPMigrate(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()

	ctx := context.Background()
	// This come from iam_policy tests to use policies in tests
	project, iamAPIKey, terminateFakeSideProject, err := iam.CreateFakeIAMManager(tt)
	require.NoError(t, err)

	// This is the provider factory that will use the temporary project
	providerFactories := iam.FakeSideProjectProviders(ctx, tt, project, iamAPIKey)

	// Goal of this test is to check that an IP will not get detached if moved from ip_id to ip_ids
	// Between the two steps we will create an API key that cannot update the IP,
	// it should fail if the provider tries to detach
	temporaryAccessKey := ""
	temporarySecretKey := ""
	customProviderFactory := map[string]func() (*schema.Provider, error){
		"scaleway": func() (*schema.Provider, error) {
			m, err := meta.NewMeta(context.Background(), &meta.Config{
				ProviderSchema:   nil,
				TerraformVersion: "terraform-tests",
				HTTPClient:       tt.Meta.HTTPClient(),
				ForceAccessKey:   temporaryAccessKey,
				ForceSecretKey:   temporarySecretKey,
			})
			if err != nil {
				return nil, err
			}
			return provider.Provider(&provider.Config{Meta: m})(), nil
		},
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			func(_ *terraform.State) error {
				return terminateFakeSideProject()
			},
			testAccCheckScalewayInstanceServerDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				ProviderFactories: providerFactories,
				Config: fmt.Sprintf(`
					resource "scaleway_instance_ip" "ip" {}

					resource "scaleway_instance_server" "main" {
						ip_id = scaleway_instance_ip.ip.id
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
					}

					resource "scaleway_iam_application" "app" {
						name = "tf_tests_instance_server_ipmigrate"
					}

					resource "scaleway_iam_policy" "policy" {
						application_id = scaleway_iam_application.app.id
						rule {
							permission_set_names = ["InstancesReadOnly"]
							organization_id = %[1]q
						}
						rule {
							permission_set_names = ["ProjectReadOnly", "IAMReadOnly"]
							organization_id = %[1]q
						}
					}

					resource "scaleway_iam_api_key" "key" {
						application_id = scaleway_iam_application.app.id
					}`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "routed_ip_enabled", "false"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "public_ips.#", "1"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["scaleway_iam_api_key.key"]
						if !ok {
							return fmt.Errorf("resource was not found: %s", "scaleway_iam_api_key.key")
						}
						temporaryAccessKey = rs.Primary.Attributes["access_key"]
						temporarySecretKey = rs.Primary.Attributes["secret_key"]

						return nil
					},
				),
			},
			{
				ProviderFactories: customProviderFactory,
				// With migration supported, this should make no changes
				// This is validated because we cannot add a nat IP to ip_ids
				// This would fail if not moved from ip_id to ip_ids
				Config: fmt.Sprintf(`
					resource "scaleway_instance_ip" "ip" {
						type = "nat"
					}

					resource "scaleway_instance_server" "main" {
						ip_ids = [scaleway_instance_ip.ip.id]
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
					}

					resource "scaleway_iam_application" "app" {
						name = "tf_tests_instance_server_ipmigrate"
					}

					resource "scaleway_iam_policy" "policy" {
						application_id = scaleway_iam_application.app.id
						rule {
							permission_set_names = ["InstancesReadOnly"]
							organization_id = %[1]q
						}
						rule {
							permission_set_names = ["ProjectReadOnly", "IAMReadOnly"]
							organization_id = %[1]q
						}
					}

					resource "scaleway_iam_api_key" "key" {
						application_id = scaleway_iam_application.app.id
					}`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICsExists(tt, "scaleway_instance_server.main"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "public_ips.#", "1"),
				),
			},
			{
				ProviderFactories: tt.ProviderFactories,
				// Last step with default api key to remove resources
				Config: fmt.Sprintf(`
					resource "scaleway_instance_ip" "ip" {
						type = "nat"
					}

					resource "scaleway_instance_server" "main" {
						ip_ids = [scaleway_instance_ip.ip.id]
						image = "ubuntu_jammy"
						type  = "PRO2-XXS"
						state = "stopped"
					}

					resource "scaleway_iam_application" "app" {
						name = "tf_tests_instance_server_ipmigrate"
					}

					resource "scaleway_iam_policy" "policy" {
						application_id = scaleway_iam_application.app.id
						rule {
							permission_set_names = ["InstancesReadOnly"]
							organization_id = %[1]q
						}
						rule {
							permission_set_names = ["ProjectReadOnly", "IAMReadOnly"]
							organization_id = %[1]q
						}
					}

					resource "scaleway_iam_api_key" "key" {
						application_id = scaleway_iam_application.app.id
					}`, project.OrganizationID),
			},
		},
	})
}

func TestAccScalewayInstanceServer_BlockExternal(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_block_volume" "volume" {
						iops = 5000
						size_in_gb = 10
					}

					resource "scaleway_instance_server" "main" {
						image = "ubuntu_jammy"
						type  = "PLAY2-PICO"
						additional_volume_ids = [scaleway_block_volume.volume.id]
					}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "type", "PLAY2-PICO"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "additional_volume_ids.#", "1"),
					resource.TestCheckResourceAttrPair("scaleway_instance_server.main", "additional_volume_ids.0", "scaleway_block_volume.volume", "id"),
				),
			},
			{
				Config: `
					resource "scaleway_block_volume" "volume" {
						iops = 5000
						size_in_gb = 10
					}

					resource "scaleway_instance_server" "main" {
						image = "ubuntu_jammy"
						type  = "PLAY2-PICO"
					}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "type", "PLAY2-PICO"),
					resource.TestCheckResourceAttr("scaleway_instance_server.main", "additional_volume_ids.#", "0"),
				),
			},
		},
	})
}
