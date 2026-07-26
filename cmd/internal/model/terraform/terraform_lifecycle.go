package terraform

type TerraformLifecycle struct {
	Type         string           `hcl:"type,label"`
	Name         string           `hcl:"name,label"`
	Id           *string          `hcl:"id"`
	SpaceId      *string          `hcl:"space_id"`
	Count        *string          `hcl:"count"`
	ResourceName string           `hcl:"name"`
	Description  *string          `hcl:"description"`
	Phase        []TerraformPhase `hcl:"phase,block"`
}

type TerraformPhase struct {
	AutomaticDeploymentTargets         []string `hcl:"automatic_deployment_targets"`
	OptionalDeploymentTargets          []string `hcl:"optional_deployment_targets"`
	Name                               *string  `hcl:"name"`
	IsOptionalPhase                    bool     `hcl:"is_optional_phase"`
	MinimumEnvironmentsBeforePromotion int      `hcl:"minimum_environments_before_promotion"`
}
