package terraform

type TerraformCommunityStepTemplateData struct {
	Type         string  `hcl:"type,label"`
	Name         string  `hcl:"name,label"`
	Id           *string `hcl:"id"`
	ResourceName *string `hcl:"name"`
	Website      *string `hcl:"website"`
}
