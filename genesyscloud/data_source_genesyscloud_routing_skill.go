package genesyscloud

import (
	"context"
	"fmt"
	"terraform-provider-genesyscloud/genesyscloud/provider"
	"terraform-provider-genesyscloud/genesyscloud/util"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v121/platformclientv2"
)

func dataSourceRoutingSkill() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for Genesys Cloud Routing Skills. Select a skill by name.",
		ReadContext: provider.ReadWithPooledClient(dataSourceRoutingSkillRead),
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Skill name.",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}

func dataSourceRoutingSkillRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	const pageSize = 100
	var pageCount int

	sdkConfig := m.(*provider.ProviderMeta).ClientConfig
	routingAPI := platformclientv2.NewRoutingApiWithConfig(sdkConfig)
	name := d.Get("name").(string)

	skills, _, getErr := routingAPI.GetRoutingSkills(pageSize, 1, name, nil)
	if getErr != nil {
		return diag.Errorf("error requesting skill %s: %s", name, getErr)
	}
	pageCount = *skills.PageCount

	// Find first non-deleted skill by name. Retry in case new skill is not yet indexed by search
	return util.WithRetries(ctx, 15*time.Second, func() *retry.RetryError {
		for pageNum := 1; pageNum <= pageCount; pageNum++ {
			const pageSize = 100
			skills, _, getErr := routingAPI.GetRoutingSkills(pageSize, pageNum, name, nil)
			if getErr != nil {
				return retry.NonRetryableError(fmt.Errorf("error requesting skill %s: %s", name, getErr))
			}

			if skills.Entities == nil || len(*skills.Entities) == 0 {
				return retry.RetryableError(fmt.Errorf("no routing skills found with name %s", name))
			}

			for _, skill := range *skills.Entities {
				if skill.Name != nil && *skill.Name == name &&
					skill.State != nil && *skill.State != "deleted" {
					d.SetId(*skill.Id)
					return nil
				}
			}
		}
		return retry.RetryableError(fmt.Errorf("no routing skills found with name %s", name))
	})
}
