package telephony_providers_edges_extension_pool

import (
	"context"
	"fmt"
	"log"
	"terraform-provider-genesyscloud/genesyscloud/provider"
	"terraform-provider-genesyscloud/genesyscloud/util"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"terraform-provider-genesyscloud/genesyscloud/consistency_checker"

	resourceExporter "terraform-provider-genesyscloud/genesyscloud/resource_exporter"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v121/platformclientv2"
)

func getAllExtensionPools(ctx context.Context, clientConfig *platformclientv2.Configuration) (resourceExporter.ResourceIDMetaMap, diag.Diagnostics) {
	resources := make(resourceExporter.ResourceIDMetaMap)
	extensionPoolProxy := getExtensionPoolProxy(clientConfig)
	extensionPools, err := extensionPoolProxy.getAllExtensionPools(ctx)
	if err != nil {
		return nil, diag.Errorf("failed to get all extension pools: %s", err)
	}
	if extensionPools != nil {
		for _, extensionPool := range *extensionPools {
			resources[*extensionPool.Id] = &resourceExporter.ResourceMeta{Name: *extensionPool.StartNumber}
		}
	}

	return resources, nil
}

func createExtensionPool(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	startNumber := d.Get("start_number").(string)
	endNumber := d.Get("end_number").(string)
	description := d.Get("description").(string)
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	extensionPoolProxy := getExtensionPoolProxy(sdkConfig)

	log.Printf("Creating Extension pool %s", startNumber)
	extensionPool, _, err := extensionPoolProxy.createExtensionPool(ctx, platformclientv2.Extensionpool{
		StartNumber: &startNumber,
		EndNumber:   &endNumber,
		Description: &description,
	})
	if err != nil {
		return diag.Errorf("Failed to create Extension pool %s: %s", startNumber, err)
	}

	d.SetId(*extensionPool.Id)
	log.Printf("Created Extension pool %s %s", startNumber, *extensionPool.Id)
	return readExtensionPool(ctx, d, meta)
}

func readExtensionPool(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	extensionPoolProxy := getExtensionPoolProxy(sdkConfig)

	log.Printf("Reading Extension pool %s", d.Id())
	return util.WithRetriesForRead(ctx, d, func() *retry.RetryError {
		extensionPool, resp, getErr := extensionPoolProxy.getExtensionPool(ctx, d.Id())
		if getErr != nil {
			if util.IsStatus404(resp) {
				return retry.RetryableError(fmt.Errorf("Failed to read Extension pool %s: %s", d.Id(), getErr))
			}
			return retry.NonRetryableError(fmt.Errorf("Failed to read Extension pool %s: %s", d.Id(), getErr))
		}

		if extensionPool.State != nil && *extensionPool.State == "deleted" {
			d.SetId("")
			return nil
		}

		cc := consistency_checker.NewConsistencyCheck(ctx, d, meta, ResourceTelephonyExtensionPool())
		d.Set("start_number", *extensionPool.StartNumber)
		d.Set("end_number", *extensionPool.EndNumber)

		if extensionPool.Description != nil {
			d.Set("description", *extensionPool.Description)
		} else {
			d.Set("description", nil)
		}

		log.Printf("Read Extension pool %s %s", d.Id(), *extensionPool.StartNumber)
		return cc.CheckState()
	})
}

func updateExtensionPool(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	startNumber := d.Get("start_number").(string)
	endNumber := d.Get("end_number").(string)
	description := d.Get("description").(string)

	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	extensionPoolProxy := getExtensionPoolProxy(sdkConfig)
	extensionPoolBody := platformclientv2.Extensionpool{
		StartNumber: &startNumber,
		EndNumber:   &endNumber,
		Description: &description,
	}
	log.Printf("Updating Extension pool %s", d.Id())
	if _, _, err := extensionPoolProxy.updateExtensionPool(ctx, d.Id(), extensionPoolBody); err != nil {
		return diag.Errorf("Error updating Extension pool %s: %s", startNumber, err)
	}
	log.Printf("Updated Extension pool %s", d.Id())
	return readExtensionPool(ctx, d, meta)
}

func deleteExtensionPool(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	startNumber := d.Get("start_number").(string)
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	extensionPoolProxy := getExtensionPoolProxy(sdkConfig)
	log.Printf("Deleting Extension pool with starting number %s", startNumber)
	if _, err := extensionPoolProxy.deleteExtensionPool(ctx, d.Id()); err != nil {
		return diag.Errorf("failed to delete Extension pool with starting number %s: %s", startNumber, err)
	}
	return util.WithRetries(ctx, 30*time.Second, func() *retry.RetryError {
		extensionPool, resp, err := extensionPoolProxy.getExtensionPool(ctx, d.Id())
		if err != nil {
			if util.IsStatus404(resp) {
				// Extension pool deleted
				log.Printf("Deleted Extension pool %s", d.Id())
				return nil
			}
			return retry.NonRetryableError(fmt.Errorf("error deleting Extension pool %s: %s", d.Id(), err))
		}
		if extensionPool.State != nil && *extensionPool.State == "deleted" {
			// Extension pool deleted
			log.Printf("Deleted Extension pool %s", d.Id())
			return nil
		}
		return retry.RetryableError(fmt.Errorf("extension pool %s still exists", d.Id()))
	})
}
