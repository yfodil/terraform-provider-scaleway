package scaleway

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	applesilicon "github.com/scaleway/scaleway-sdk-go/api/applesilicon/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality/zonal"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/meta"
)

const (
	defaultAppleSiliconServerTimeout       = 2 * time.Minute
	defaultAppleSiliconServerRetryInterval = 5 * time.Second
)

// asAPIWithZone returns a new apple silicon API and the zone
func asAPIWithZone(d *schema.ResourceData, m interface{}) (*applesilicon.API, scw.Zone, error) {
	asAPI := applesilicon.NewAPI(meta.ExtractScwClient(m))

	zone, err := meta.ExtractZone(d, m)
	if err != nil {
		return nil, "", err
	}
	return asAPI, zone, nil
}

// AsAPIWithZoneAndID returns an apple silicon API with zone and ID extracted from the state
func AsAPIWithZoneAndID(m interface{}, id string) (*applesilicon.API, scw.Zone, string, error) {
	asAPI := applesilicon.NewAPI(meta.ExtractScwClient(m))

	zone, ID, err := zonal.ParseID(id)
	if err != nil {
		return nil, "", "", err
	}
	return asAPI, zone, ID, nil
}

func waitForAppleSiliconServer(ctx context.Context, api *applesilicon.API, zone scw.Zone, serverID string, timeout time.Duration) (*applesilicon.Server, error) {
	retryInterval := defaultAppleSiliconServerRetryInterval
	if transport.DefaultWaitRetryInterval != nil {
		retryInterval = *transport.DefaultWaitRetryInterval
	}

	server, err := api.WaitForServer(&applesilicon.WaitForServerRequest{
		ServerID:      serverID,
		Zone:          zone,
		Timeout:       scw.TimeDurationPtr(timeout),
		RetryInterval: &retryInterval,
	}, scw.WithContext(ctx))

	return server, err
}
