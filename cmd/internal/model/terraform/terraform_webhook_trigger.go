package terraform

// TerraformWebhookTrigger represents the octopusdeploy_webhook_trigger resource.
// https://registry.terraform.io/providers/OctopusDeploy/octopusdeploy/latest/docs/resources/webhook_trigger
type TerraformWebhookTrigger struct {
	Type             string                                   `hcl:"type,label"`
	Name             string                                   `hcl:"name,label"`
	Count            *string                                  `hcl:"count"`
	Id               *string                                  `hcl:"id"`
	SpaceId          *string                                  `hcl:"space_id"`
	ResourceName     string                                   `hcl:"name"`
	Description      *string                                  `hcl:"description"`
	ProjectId        string                                   `hcl:"project_id"`
	IsDisabled       *bool                                    `hcl:"is_disabled"`
	Secret           *string                                  `hcl:"secret"`
	RequireApiKey    *bool                                    `hcl:"require_api_key"`
	TenantIds        []string                                 `hcl:"tenant_ids"`
	RunRunbookAction *TerraformWebhookTriggerRunRunbookAction `hcl:"run_runbook_action"`
}

type TerraformWebhookTriggerRunRunbookAction struct {
	RunbookId            string   `cty:"runbook_id"`
	TargetEnvironmentIds []string `cty:"target_environment_ids"`
}
