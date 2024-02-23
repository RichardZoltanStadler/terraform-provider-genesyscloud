package flow_outcome

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"terraform-provider-genesyscloud/genesyscloud/provider"
	"terraform-provider-genesyscloud/genesyscloud/util"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

/*
   The data_source_genesyscloud_flow_outcome.go contains the data source implementation
   for the resource.
*/

// dataSourceFlowOutcomeRead retrieves by name the id in question
func dataSourceFlowOutcomeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := newFlowOutcomeProxy(sdkConfig)

	name := d.Get("name").(string)

	return util.WithRetries(ctx, 15*time.Second, func() *retry.RetryError {
		flowOutcomeId, retryable, err := proxy.getFlowOutcomeIdByName(ctx, name)

		if err != nil && !retryable {
			return retry.NonRetryableError(fmt.Errorf("Error searching flow outcome %s: %s", name, err))
		}

		if retryable {
			return retry.RetryableError(fmt.Errorf("No flow outcome found with name %s", name))
		}

		d.SetId(flowOutcomeId)
		return nil
	})
}
