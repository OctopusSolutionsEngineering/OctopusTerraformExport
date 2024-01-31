package terraform

type TerraformLifecycle struct {
	Type                    string           `hcl:"type,label"`
	Name                    string           `hcl:"name,label"`
	Count                   *string          `hcl:"count"`
	ResourceName            string           `hcl:"name"`
	Description             *string          `hcl:"description"`
	Phase                   []TerraformPhase `hcl:"phase,block"`
	ReleaseRetentionPolicy  *TerraformPolicy `hcl:"release_retention_policy,block"`
	TentacleRetentionPolicy *TerraformPolicy `hcl:"tentacle_retention_policy,block"`
}

type TerraformPhase struct {
	AutomaticDeploymentTargets         []string         `hcl:"automatic_deployment_targets"`
	OptionalDeploymentTargets          []string         `hcl:"optional_deployment_targets"`
	Name                               *string          `hcl:"name"`
	IsOptionalPhase                    bool             `hcl:"is_optional_phase"`
	MinimumEnvironmentsBeforePromotion int              `hcl:"minimum_environments_before_promotion"`
	ReleaseRetentionPolicy             *TerraformPolicy `hcl:"release_retention_policy,block"`
	TentacleRetentionPolicy            *TerraformPolicy `hcl:"tentacle_retention_policy,block"`
}

type TerraformPolicy struct {
	QuantityToKeep    int    `hcl:"quantity_to_keep"`
	ShouldKeepForever bool   `hcl:"should_keep_forever"`
	Unit              string `hcl:"unit"`
}
