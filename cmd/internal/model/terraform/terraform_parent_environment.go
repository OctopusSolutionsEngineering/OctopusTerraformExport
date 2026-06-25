package terraform

type TerraformParentEnvironment struct {
	Type                        string                                `hcl:"type,label"`
	Name                        string                                `hcl:"name,label"`
	Count                       *string                               `hcl:"count"`
	SpaceId                     *string                               `hcl:"space_id"`
	Id                          *string                               `hcl:"id"`
	ResourceName                string                                `hcl:"name"`
	Description                 *string                               `hcl:"description"`
	UseGuidedFailure            bool                                  `hcl:"use_guided_failure"`
	AutomaticDeprovisioningRule *TerraformAutomaticDeprovisioningRule `hcl:"automatic_deprovisioning_rule,block"`
}

type TerraformAutomaticDeprovisioningRule struct {
	Days  int `hcl:"days"`
	Hours int `hcl:"hours"`
}
