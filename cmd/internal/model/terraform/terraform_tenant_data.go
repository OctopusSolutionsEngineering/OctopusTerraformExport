package terraform

type TerraformTenantData struct {
	Type        string                          `hcl:"type,label"`
	Name        string                          `hcl:"name,label"`
	Ids         []string                        `hcl:"ids"`
	PartialName string                          `hcl:"partial_name"`
	Skip        int                             `hcl:"skip"`
	Take        int                             `hcl:"take"`
	ProjectId   string                          `hcl:"project_id"`
	Tags        []string                        `hcl:"tags"`
	Lifecycle   *TerraformLifecycleMetaArgument `hcl:"lifecycle,block"`
}
