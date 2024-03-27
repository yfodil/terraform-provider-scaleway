package scaleway

import (
	billing "github.com/scaleway/scaleway-sdk-go/api/billing/v2beta1"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/meta"
)

// billingAPI returns a new billing API.
func billingAPI(m interface{}) *billing.API {
	return billing.NewAPI(meta.ExtractScwClient(m))
}
