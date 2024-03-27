package scaleway

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	flexibleip "github.com/scaleway/scaleway-sdk-go/api/flexibleip/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality/zonal"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/meta"
)

const (
	defaultFlexibleIPTimeout = 1 * time.Minute
	retryFlexibleIPInterval  = 5 * time.Second
)

// fipAPIWithZone returns an lb API WITH zone for a Create request
func fipAPIWithZone(d *schema.ResourceData, m interface{}) (*flexibleip.API, scw.Zone, error) {
	flexibleipAPI := flexibleip.NewAPI(meta.ExtractScwClient(m))

	zone, err := meta.ExtractZone(d, m)
	if err != nil {
		return nil, "", err
	}
	return flexibleipAPI, zone, nil
}

// FipAPIWithZoneAndID returns an flexibleip API with zone and ID extracted from the state
func FipAPIWithZoneAndID(m interface{}, id string) (*flexibleip.API, scw.Zone, string, error) {
	fipAPI := flexibleip.NewAPI(meta.ExtractScwClient(m))

	zone, ID, err := zonal.ParseID(id)
	if err != nil {
		return nil, "", "", err
	}
	return fipAPI, zone, ID, nil
}

func waitFlexibleIP(ctx context.Context, api *flexibleip.API, zone scw.Zone, id string, timeout time.Duration) (*flexibleip.FlexibleIP, error) {
	retryInterval := retryFlexibleIPInterval
	if transport.DefaultWaitRetryInterval != nil {
		retryInterval = *transport.DefaultWaitRetryInterval
	}

	return api.WaitForFlexibleIP(&flexibleip.WaitForFlexibleIPRequest{
		FipID:         id,
		Zone:          zone,
		Timeout:       scw.TimeDurationPtr(timeout),
		RetryInterval: &retryInterval,
	}, scw.WithContext(ctx))
}

func flattenFlexibleIPMacAddress(mac *flexibleip.MACAddress) interface{} {
	if mac == nil {
		return nil
	}
	return []map[string]interface{}{
		{
			"id":          mac.ID,
			"mac_address": mac.MacAddress,
			"mac_type":    mac.MacType,
			"status":      mac.Status,
			"created_at":  types.FlattenTime(mac.CreatedAt),
			"updated_at":  types.FlattenTime(mac.UpdatedAt),
			"zone":        mac.Zone,
		},
	}
}

func expandServerIDs(data interface{}) []string {
	expandedIDs := make([]string, 0, len(data.([]interface{})))
	for _, s := range data.([]interface{}) {
		if s == nil {
			s = ""
		}
		expandedID := locality.ExpandID(s.(string))
		expandedIDs = append(expandedIDs, expandedID)
	}
	return expandedIDs
}
