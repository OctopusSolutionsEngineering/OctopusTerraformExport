package terraform

type TerraformGitCredentialData struct {
	Type         string                          `hcl:"type,label"`
	Name         string                          `hcl:"name,label"`
	Id           *string                         `hcl:"id"`
	ResourceName string                          `hcl:"name"`
	Skip         int                             `hcl:"skip"`
	Take         int                             `hcl:"take"`
	Lifecycle    *TerraformLifecycleMetaArgument `hcl:"lifecycle,block"`
}
