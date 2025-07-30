package terraform

// TerraformStepTemplate represents a step template in Terraform
// https://registry.terraform.io/providers/OctopusDeployLabs/octopusdeploy/latest/docs/resources/step_template
type TerraformStepTemplate struct {
	Type                      string                           `hcl:"type,label"`
	Name                      string                           `hcl:"name,label"`
	Count                     *string                          `hcl:"count"`
	ActionType                string                           `hcl:"action_type"`
	SpaceId                   *string                          `hcl:"space_id"`
	ResourceName              string                           `hcl:"name"`
	Description               *string                          `hcl:"description"`
	StepPackageId             string                           `hcl:"step_package_id"`
	CommunityActionTemplateId *string                          `hcl:"community_action_template_id"`
	Packages                  []TerraformStepTemplatePackage   `hcl:"packages"`
	Parameters                []TerraformStepTemplateParameter `hcl:"parameters"`
	Properties                map[string]string                `hcl:"properties"`
}

type TerraformStepTemplateParameter struct {
	Id                    string            `cty:"id"`
	Name                  string            `cty:"name"`
	Label                 *string           `cty:"label"`
	HelpText              *string           `cty:"help_text"`
	DefaultValue          *string           `cty:"default_value"`
	DefaultSensitiveValue *string           `cty:"default_sensitive_value"`
	DisplaySettings       map[string]string `cty:"display_settings"`
}

type TerraformStepTemplatePackage struct {
	Name                    string                                 `cty:"name"`
	PackageID               *string                                `cty:"package_id"`
	AcquisitionLocation     *string                                `cty:"acquisition_location"`
	ExtractDuringDeployment *bool                                  `cty:"extract_during_deployment"`
	FeedId                  string                                 `cty:"feed_id"`
	Properties              TerraformStepTemplatePackageProperties `cty:"properties"`
}

type TerraformStepTemplatePackageProperties struct {
	SelectionMode        string `cty:"selection_mode"`
	Extract              string `cty:"selection_mode"`
	PackageParameterName string `cty:"package_parameter_name"`
	Purpose              string `cty:"purpose"`
}
