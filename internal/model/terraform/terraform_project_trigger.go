package terraform

type TerraformProjectTrigger struct {
	Type            string   `hcl:"type,label"`
	Name            string   `hcl:"name,label"`
	ResourceName    string   `hcl:"name"`
	ProjectId       string   `hcl:"project_id"`
	EventCategories []string `hcl:"event_categories"`
	EnvironmentIds  []string `hcl:"environment_ids"`
	EventGroups     []string `hcl:"event_groups"`
	Roles           []string `hcl:"roles"`
	ShouldRedeploy  bool     `hcl:"should_redeploy"`
	Id              *string  `hcl:"id"`
}
