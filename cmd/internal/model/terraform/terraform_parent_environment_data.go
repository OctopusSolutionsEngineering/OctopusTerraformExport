package terraform

type TerraformParentEnvironmentData struct {
	Type         string                          `hcl:"type,label"`
	Name         string                          `hcl:"name,label"`
	Ids          []string                        `hcl:"ids"`
	PartialName  string                          `hcl:"partial_name"`
	ResourceName string                          `hcl:"name"`
	Skip         int                             `hcl:"skip"`
	Take         int                             `hcl:"take"`
	Lifecycle    *TerraformLifecycleMetaArgument `hcl:"lifecycle,block"`
}
