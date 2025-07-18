package terraform

type TerraformProcess struct {
	Type      string  `hcl:"type,label"`
	Name      string  `hcl:"name,label"`
	Count     *string `hcl:"count"`
	Id        *string `hcl:"id"`
	SpaceId   *string `hcl:"space_id"`
	ProjectId *string `hcl:"project_id"`
	RunbookId *string `hcl:"runbook_id"`
}
