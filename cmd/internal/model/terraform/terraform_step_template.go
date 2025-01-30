package terraform

// TerraformStepTemplate represents a step template in Terraform
// https://registry.terraform.io/providers/OctopusDeployLabs/octopusdeploy/latest/docs/resources/step_template
type TerraformStepTemplate struct {
	Type                      string              `hcl:"type,label"`
	Name                      string              `hcl:"name,label"`
	Count                     *string             `hcl:"count"`
	ActionType                string              `hcl:"action_type"`
	SpaceId                   *string             `hcl:"space_id"`
	ResourceName              string              `hcl:"name"`
	Description               *string             `hcl:"description"`
	StepPackageId             string              `hcl:"step_package_id"`
	CommunityActionTemplateId *string             `hcl:"community_action_template_id"`
	Packages                  []TerraformPackage  `hcl:"packages,block"`
	DisplaySettings           []TerraformTemplate `hcl:"display_settings,block"`
	Properties                map[string]string   `hcl:"properties"`
}
