package terraform

type TerraformChannelData struct {
	Type        string                          `hcl:"type,label"`
	Name        string                          `hcl:"name,label"`
	Ids         []string                        `hcl:"ids"`
	PartialName string                          `hcl:"partial_name"`
	ProjectId   *string                         `hcl:"project_id"`
	Skip        int                             `hcl:"skip"`
	Take        int                             `hcl:"take"`
	Lifecycle   *TerraformLifecycleMetaArgument `hcl:"lifecycle,block"`
}
