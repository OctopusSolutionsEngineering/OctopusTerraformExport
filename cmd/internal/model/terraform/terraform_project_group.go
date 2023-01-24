package terraform

type TerraformProjectGroup struct {
	Type              string  `hcl:"type,label"`
	Name              string  `hcl:"name,label"`
	ResourceName      string  `hcl:"name"`
	Description       *string `hcl:"description"`
	RetentionPolicyId *string `hcl:"retention_policy_id"`
	SpaceId           *string `hcl:"space_id"`
}
