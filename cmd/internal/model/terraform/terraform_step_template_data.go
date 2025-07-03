package terraform

// TerraformStepTemplateData represents a step template in Terraform
// https://registry.terraform.io/providers/OctopusDeploy/octopusdeploy/latest/docs/data-sources/step_template
type TerraformStepTemplateData struct {
	Type    string  `hcl:"type,label"`
	Name    string  `hcl:"name,label"`
	SpaceId *string `hcl:"space_id"`
	Id      string  `hcl:"id"`
}
