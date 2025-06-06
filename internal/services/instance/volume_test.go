package instance_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/acctest"
	instancetestfuncs "github.com/scaleway/terraform-provider-scaleway/v2/internal/services/instance/testfuncs"
)

func TestAccVolume_Basic(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      instancetestfuncs.IsVolumeDestroyed(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "test" {
						type       = "l_ssd"
						size_in_gb = 20
						tags = ["test-terraform"]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					instancetestfuncs.IsVolumePresent(tt, "scaleway_instance_volume.test"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.test", "size_in_gb", "20"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.test", "tags.0", "test-terraform"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_volume" "test" {
						type       = "l_ssd"
						name       = "terraform-test"
						size_in_gb = 20
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					instancetestfuncs.IsVolumePresent(tt, "scaleway_instance_volume.test"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.test", "name", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.test", "size_in_gb", "20"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.test", "tags.#", "0"),
				),
			},
		},
	})
}

func TestAccVolume_DifferentNameGenerated(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      instancetestfuncs.IsVolumeDestroyed(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "test" {
						type       = "l_ssd"
						size_in_gb = 20
					}
				`,
			},
			{
				Config: `
					resource "scaleway_instance_volume" "test" {
						type       = "l_ssd"
						size_in_gb = 20
					}
				`,
			},
		},
	})
}

func TestAccVolume_ResizeBlock(t *testing.T) {
	t.Skip("Resource \"scaleway_instance_volume\" is depracated for block volumes")

	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      instancetestfuncs.IsVolumeDestroyed(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "b_ssd"
						size_in_gb = 20
					}`,
				Check: resource.ComposeTestCheckFunc(
					instancetestfuncs.IsVolumePresent(tt, "scaleway_instance_volume.main"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.main", "size_in_gb", "20"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "b_ssd"
						size_in_gb = 30
					}`,
				Check: resource.ComposeTestCheckFunc(
					instancetestfuncs.IsVolumePresent(tt, "scaleway_instance_volume.main"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.main", "size_in_gb", "30"),
				),
			},
		},
	})
}

func TestAccVolume_ResizeNotBlock(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      instancetestfuncs.IsVolumeDestroyed(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "l_ssd"
						size_in_gb = 20
					}`,
				Check: resource.ComposeTestCheckFunc(
					instancetestfuncs.IsVolumePresent(tt, "scaleway_instance_volume.main"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.main", "size_in_gb", "20"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "l_ssd"
						size_in_gb = 30
					}`,
				ExpectError: regexp.MustCompile("only block volume can be resized"),
			},
		},
	})
}

func TestAccVolume_CannotResizeBlockDown(t *testing.T) {
	t.Skip("Resource \"scaleway_instance_volume\" is depracated for block volumes")

	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      instancetestfuncs.IsVolumeDestroyed(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "b_ssd"
						size_in_gb = 20
					}`,
			},
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "b_ssd"
						size_in_gb = 10
					}`,
				ExpectError: regexp.MustCompile("block volumes cannot be resized down"),
			},
		},
	})
}

func TestAccVolume_Scratch(t *testing.T) {
	tt := acctest.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      instancetestfuncs.IsVolumeDestroyed(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "scratch"
						size_in_gb = 20
					}`,
			},
		},
	})
}
