package terraform

type TerraformWorkerPoolData struct {
	Type         string                          `hcl:"type,label"`
	Name         string                          `hcl:"name,label"`
	Count        *string                         `hcl:"count"`
	ResourceName *string                         `hcl:"name"`
	Ids          []string                        `hcl:"ids"`
	PartialName  *string                         `hcl:"partial_name"`
	Skip         int                             `hcl:"skip"`
	Take         int                             `hcl:"take"`
	Lifecycle    *TerraformLifecycleMetaArgument `hcl:"lifecycle,block"`
}
