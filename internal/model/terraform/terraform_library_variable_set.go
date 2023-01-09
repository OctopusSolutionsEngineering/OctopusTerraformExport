package terraform

type TerraformLibraryVariableSet struct {
	Type         string              `hcl:"type,label"`
	Name         string              `hcl:"name,label"`
	SpaceId      *string             `hcl:"space_id"`
	ResourceName string              `hcl:"name"`
	Description  *string             `hcl:"description"`
	Template     []TerraformTemplate `hcl:"template,block"`
}
