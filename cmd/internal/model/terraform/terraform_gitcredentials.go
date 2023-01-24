package terraform

type TerraformGitCredentials struct {
	Type         string  `hcl:"type,label"`
	Name         string  `hcl:"name,label"`
	Id           *string `hcl:"id"`
	Description  *string `hcl:"description"`
	ResourceName string  `hcl:"name"`
	ResourceType string  `hcl:"type"`
	Username     string  `hcl:"username"`
	Password     string  `hcl:"password"`
}
