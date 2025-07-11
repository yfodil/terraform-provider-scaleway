package applesilicon

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	applesilicon "github.com/scaleway/scaleway-sdk-go/api/applesilicon/v1alpha1"
	ipamAPI "github.com/scaleway/scaleway-sdk-go/api/ipam/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/httperrors"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality/zonal"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/account"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/ipam"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"
)

func ResourceServer() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceAppleSiliconServerCreate,
		ReadContext:   ResourceAppleSiliconServerRead,
		UpdateContext: ResourceAppleSiliconServerUpdate,
		DeleteContext: ResourceAppleSiliconServerDelete,
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultAppleSiliconServerTimeout),
			Default: schema.DefaultTimeout(defaultAppleSiliconServerTimeout),
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the server",
				Computed:    true,
				Optional:    true,
			},
			"type": {
				Type:        schema.TypeString,
				Description: "Type of the server",
				Required:    true,
				ForceNew:    true,
			},
			"enable_vpc": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether or not to enable VPC access",
			},
			"commitment": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "duration_24h",
				Description:      "The commitment period of the server",
				ValidateDiagFunc: verify.ValidateEnum[applesilicon.CommitmentType](),
			},
			"public_bandwidth": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "The public bandwidth of the server in bits per second",
			},
			"private_network": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The private networks to attach to the server",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:             schema.TypeString,
							Description:      "The private network ID",
							Required:         true,
							ValidateDiagFunc: verify.IsUUIDorUUIDWithLocality(),
							StateFunc: func(i any) string {
								return locality.ExpandID(i.(string))
							},
						},
						"ipam_ip_ids": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: verify.IsUUIDorUUIDWithLocality(),
							},
							Description: "List of IPAM IP IDs to attach to the server",
						},
						// computed
						"vlan": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The VLAN ID associated to the private network",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The private network status",
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The date and time of the creation of the private network",
						},
						"updated_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The date and time of the last update of the private network",
						},
					},
				},
			},
			// Computed
			"ip": {
				Type:        schema.TypeString,
				Description: "IPv4 address of the server",
				Computed:    true,
			},
			"private_ips": {
				Type:        schema.TypeList,
				Computed:    true,
				Optional:    true,
				Description: "List of private IPv4 and IPv6 addresses associated with the server",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the IP address resource",
						},
						"address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The private IP address",
						},
					},
				},
			},
			"vnc_url": {
				Type:        schema.TypeString,
				Description: "VNC url use to connect remotely to the desktop GUI",
				Computed:    true,
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The state of the server",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the server",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the server",
			},
			"deletable_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The minimal date and time on which you can delete this server due to Apple licence",
			},
			"vpc_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The VPC status of the server",
			},
			"password": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The password of the server",
			},
			"username": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The username of the server",
			},

			// Common
			"zone":            zonal.Schema(),
			"organization_id": account.OrganizationIDSchema(),
			"project_id":      account.ProjectIDSchema(),
		},
	}
}

func ResourceAppleSiliconServerCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	asAPI, zone, err := newAPIWithZone(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	createReq := &applesilicon.CreateServerRequest{
		Name:           types.ExpandOrGenerateString(d.Get("name"), "m1"),
		Type:           d.Get("type").(string),
		ProjectID:      d.Get("project_id").(string),
		EnableVpc:      d.Get("enable_vpc").(bool),
		CommitmentType: applesilicon.CommitmentType(d.Get("commitment").(string)),
		Zone:           zone,
	}

	if bandwidth, ok := d.GetOk("public_bandwidth"); ok {
		createReq.PublicBandwidthBps = *types.ExpandUint64Ptr(bandwidth)
	}

	res, err := asAPI.CreateServer(createReq, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(zonal.NewIDString(zone, res.ID))

	_, err = waitForAppleSiliconServer(ctx, asAPI, zone, res.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	if pn, ok := d.GetOk("private_network"); ok {
		privateNetworkAPI := applesilicon.NewPrivateNetworkAPI(meta.ExtractScwClient(m))
		req := &applesilicon.PrivateNetworkAPISetServerPrivateNetworksRequest{
			Zone:                       zone,
			ServerID:                   res.ID,
			PerPrivateNetworkIpamIPIDs: expandPrivateNetworks(pn),
		}

		_, err := privateNetworkAPI.SetServerPrivateNetworks(req, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = waitForAppleSiliconPrivateNetworkServer(ctx, privateNetworkAPI, zone, res.ID, d.Timeout(schema.TimeoutCreate))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceAppleSiliconServerRead(ctx, d, m)
}

func ResourceAppleSiliconServerRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	asAPI, zone, ID, err := NewAPIWithZoneAndID(m, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	privateNetworkAPI := applesilicon.NewPrivateNetworkAPI(meta.ExtractScwClient(m))

	res, err := asAPI.GetServer(&applesilicon.GetServerRequest{
		Zone:     zone,
		ServerID: ID,
	}, scw.WithContext(ctx))
	if err != nil {
		if httperrors.Is404(err) {
			d.SetId("")

			return nil
		}

		return diag.FromErr(err)
	}

	_ = d.Set("name", res.Name)
	_ = d.Set("type", res.Type)
	_ = d.Set("state", res.Status.String())
	_ = d.Set("created_at", res.CreatedAt.Format(time.RFC3339))
	_ = d.Set("updated_at", res.UpdatedAt.Format(time.RFC3339))
	_ = d.Set("deletable_at", res.DeletableAt.Format(time.RFC3339))
	_ = d.Set("ip", res.IP.String())
	_ = d.Set("vnc_url", res.VncURL)
	_ = d.Set("vpc_status", res.VpcStatus)
	_ = d.Set("zone", res.Zone.String())
	_ = d.Set("organization_id", res.OrganizationID)
	_ = d.Set("project_id", res.ProjectID)
	_ = d.Set("password", res.SudoPassword)
	_ = d.Set("username", res.SSHUsername)
	_ = d.Set("public_bandwidth", int(res.PublicBandwidthBps))
	_ = d.Set("zone", res.Zone)

	listPrivateNetworks, err := privateNetworkAPI.ListServerPrivateNetworks(&applesilicon.PrivateNetworkAPIListServerPrivateNetworksRequest{
		Zone:     res.Zone,
		ServerID: &res.ID,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	pnRegion, err := res.Zone.Region()
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("private_network", flattenPrivateNetworks(pnRegion, listPrivateNetworks.ServerPrivateNetworks))

	privateNetworkIDs := make([]string, 0, listPrivateNetworks.TotalCount)
	for _, pn := range listPrivateNetworks.ServerPrivateNetworks {
		privateNetworkIDs = append(privateNetworkIDs, pn.PrivateNetworkID)
	}

	diags := diag.Diagnostics{}
	allPrivateIPs := make([]map[string]any, 0, listPrivateNetworks.TotalCount)
	authorized := true

	for _, privateNetworkID := range privateNetworkIDs {
		resourceType := ipamAPI.ResourceTypeAppleSiliconPrivateNic
		opts := &ipam.GetResourcePrivateIPsOptions{
			ResourceType:     &resourceType,
			PrivateNetworkID: &privateNetworkID,
			ProjectID:        &res.ProjectID,
		}

		privateIPs, err := ipam.GetResourcePrivateIPs(ctx, m, pnRegion, opts)

		switch {
		case err == nil:
			allPrivateIPs = append(allPrivateIPs, privateIPs...)
		case httperrors.Is403(err):
			authorized = false

			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Warning,
				Summary:       "Unauthorized to read server's private IPs, please check your IAM permissions",
				Detail:        err.Error(),
				AttributePath: cty.GetAttrPath("private_ips"),
			})
		default:
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Warning,
				Summary:       fmt.Sprintf("Unable to get private IP for server %q", res.Name),
				Detail:        err.Error(),
				AttributePath: cty.GetAttrPath("private_ips"),
			})
		}

		if !authorized {
			break
		}
	}

	if authorized {
		_ = d.Set("private_ips", allPrivateIPs)
	}

	return diags
}

func ResourceAppleSiliconServerUpdate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	asAPI, zone, ID, err := NewAPIWithZoneAndID(m, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	appleSilisonPrivateNetworkAPI, zonePN, err := newPrivateNetworkAPIWithZone(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	req := &applesilicon.UpdateServerRequest{
		Zone:     zone,
		ServerID: ID,
	}

	if d.HasChange("name") {
		req.Name = types.ExpandStringPtr(d.Get("name"))
	}

	if d.HasChange("commitment") {
		req.CommitmentType = &applesilicon.CommitmentTypeValue{CommitmentType: applesilicon.CommitmentType(d.Get("commitment").(string))}
	}

	if d.HasChange("enable_vpc") {
		enableVpc := d.Get("enable_vpc").(bool)
		req.EnableVpc = &enableVpc
	}

	if d.HasChange("public_bandwidth") {
		publicBandwidth := types.ExpandUint64Ptr(d.Get("public_bandwidth"))
		req.PublicBandwidthBps = publicBandwidth
	}

	_, err = asAPI.UpdateServer(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	err = waitForTerminalVPCState(ctx, asAPI, zone, ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("private_network") && d.Get("enable_vpc").(bool) {
		privateNetwork := d.Get("private_network")
		req := &applesilicon.PrivateNetworkAPISetServerPrivateNetworksRequest{
			Zone:                       zonePN,
			ServerID:                   ID,
			PerPrivateNetworkIpamIPIDs: expandPrivateNetworks(privateNetwork),
		}

		_, err := appleSilisonPrivateNetworkAPI.SetServerPrivateNetworks(req, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	_, err = waitForAppleSiliconPrivateNetworkServer(ctx, appleSilisonPrivateNetworkAPI, zone, ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceAppleSiliconServerRead(ctx, d, m)
}

func ResourceAppleSiliconServerDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	asAPI, zone, ID, err := NewAPIWithZoneAndID(m, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = detachAllPrivateNetworkFromServer(ctx, d, m, ID)
	if err != nil {
		return diag.FromErr(err)
	}

	err = asAPI.DeleteServer(&applesilicon.DeleteServerRequest{
		Zone:     zone,
		ServerID: ID,
	}, scw.WithContext(ctx))

	if err != nil && !httperrors.Is403(err) {
		return diag.FromErr(err)
	}

	return nil
}
