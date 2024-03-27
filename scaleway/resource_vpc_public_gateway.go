package scaleway

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/httperrors"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality/zonal"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
)

func ResourceScalewayVPCPublicGateway() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayVPCPublicGatewayCreate,
		ReadContext:   resourceScalewayVPCPublicGatewayRead,
		UpdateContext: resourceScalewayVPCPublicGatewayUpdate,
		DeleteContext: resourceScalewayVPCPublicGatewayDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultVPCGatewayTimeout),
			Read:    schema.DefaultTimeout(defaultVPCGatewayTimeout),
			Update:  schema.DefaultTimeout(defaultVPCGatewayTimeout),
			Delete:  schema.DefaultTimeout(defaultVPCGatewayTimeout),
			Default: schema.DefaultTimeout(defaultVPCGatewayTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "name of the gateway",
			},
			"type": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "gateway type",
				DiffSuppressFunc: diffSuppressFuncIgnoreCase,
			},
			"upstream_dns_servers": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "override the gateway's default recursive DNS servers, if DNS features are enabled",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"ip_id": {
				Type:             schema.TypeString,
				Computed:         true,
				Optional:         true,
				ForceNew:         true,
				Description:      "attach an existing IP to the gateway",
				DiffSuppressFunc: diffSuppressFuncLocality,
			},
			"tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The tags associated with public gateway",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"bastion_enabled": {
				Type:        schema.TypeBool,
				Description: "Enable SSH bastion on the gateway",
				Optional:    true,
			},
			"bastion_port": {
				Type:        schema.TypeInt,
				Description: "Port of the SSH bastion",
				Optional:    true,
				Computed:    true,
			},
			"enable_smtp": {
				Type:        schema.TypeBool,
				Description: "Enable SMTP on the gateway",
				Optional:    true,
				Computed:    true,
			},
			"project_id": projectIDSchema(),
			"zone":       zonal.Schema(),
			// Computed elements
			"organization_id": organizationIDSchema(),
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the public gateway",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the public gateway",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the public gateway",
			},
		},
	}
}

func resourceScalewayVPCPublicGatewayCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcgwAPI, zone, err := vpcgwAPIWithZone(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	req := &vpcgw.CreateGatewayRequest{
		Name:               types.ExpandOrGenerateString(d.Get("name"), "pn"),
		Type:               d.Get("type").(string),
		Tags:               types.ExpandStrings(d.Get("tags")),
		UpstreamDNSServers: types.ExpandStrings(d.Get("upstream_dns_servers")),
		ProjectID:          d.Get("project_id").(string),
		EnableBastion:      d.Get("bastion_enabled").(bool),
		Zone:               zone,
		EnableSMTP:         d.Get("enable_smtp").(bool),
	}

	if bastionPort, ok := d.GetOk("bastion_port"); ok {
		req.BastionPort = types.ExpandUint32Ptr(bastionPort.(int))
	}

	if ipID, ok := d.GetOk("ip_id"); ok {
		req.IPID = types.ExpandStringPtr(zonal.ExpandID(ipID).ID)
	}

	gateway, err := vpcgwAPI.CreateGateway(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(zonal.NewIDString(zone, gateway.ID))

	// check err waiting process
	_, err = waitForVPCPublicGateway(ctx, vpcgwAPI, zone, gateway.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayVPCPublicGatewayRead(ctx, d, m)
}

func resourceScalewayVPCPublicGatewayRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcgwAPI, zone, id, err := VpcgwAPIWithZoneAndID(m, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	gateway, err := waitForVPCPublicGateway(ctx, vpcgwAPI, zone, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if httperrors.Is404(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", gateway.Name)
	_ = d.Set("type", gateway.Type.Name)
	_ = d.Set("status", gateway.Status.String())
	_ = d.Set("organization_id", gateway.OrganizationID)
	_ = d.Set("project_id", gateway.ProjectID)
	_ = d.Set("created_at", gateway.CreatedAt.Format(time.RFC3339))
	_ = d.Set("updated_at", gateway.UpdatedAt.Format(time.RFC3339))
	_ = d.Set("zone", gateway.Zone)
	_ = d.Set("tags", gateway.Tags)
	_ = d.Set("upstream_dns_servers", gateway.UpstreamDNSServers)
	_ = d.Set("ip_id", zonal.NewID(gateway.Zone, gateway.IP.ID).String())
	_ = d.Set("bastion_enabled", gateway.BastionEnabled)
	_ = d.Set("bastion_port", int(gateway.BastionPort))
	_ = d.Set("enable_smtp", gateway.SMTPEnabled)

	return nil
}

func resourceScalewayVPCPublicGatewayUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcgwAPI, zone, id, err := VpcgwAPIWithZoneAndID(m, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	gateway, err := waitForVPCPublicGateway(ctx, vpcgwAPI, zone, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	updateRequest := &vpcgw.UpdateGatewayRequest{
		GatewayID: gateway.ID,
		Zone:      gateway.Zone,
	}

	if d.HasChanges("name") {
		updateRequest.Name = scw.StringPtr(d.Get("name").(string))
	}

	if d.HasChange("tags") {
		updateRequest.Tags = types.ExpandUpdatedStringsPtr(d.Get("tags"))
	}

	if d.HasChange("bastion_port") {
		updateRequest.BastionPort = scw.Uint32Ptr(uint32(d.Get("bastion_port").(int)))
	}

	if d.HasChange("bastion_enabled") {
		updateRequest.EnableBastion = scw.BoolPtr(d.Get("bastion_enabled").(bool))
	}

	if d.HasChange("enable_smtp") {
		updateRequest.EnableSMTP = scw.BoolPtr(d.Get("enable_smtp").(bool))
	}

	if d.HasChange("upstream_dns_servers") {
		updateRequest.UpstreamDNSServers = types.ExpandUpdatedStringsPtr(d.Get("upstream_dns_servers"))
	}

	_, err = vpcgwAPI.UpdateGateway(updateRequest, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForVPCPublicGateway(ctx, vpcgwAPI, zone, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayVPCPublicGatewayRead(ctx, d, m)
}

func resourceScalewayVPCPublicGatewayDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	vpcgwAPI, zone, id, err := VpcgwAPIWithZoneAndID(m, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForVPCPublicGateway(ctx, vpcgwAPI, zone, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	err = vpcgwAPI.DeleteGateway(&vpcgw.DeleteGatewayRequest{
		GatewayID: id,
		Zone:      zone,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForVPCPublicGateway(ctx, vpcgwAPI, zone, id, d.Timeout(schema.TimeoutDelete))
	if err != nil && !httperrors.Is404(err) {
		return diag.FromErr(err)
	}

	return nil
}
