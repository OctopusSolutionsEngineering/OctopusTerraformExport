package terraform

type TerraformLibraryVariableSet struct {
	Type         string              `hcl:"type,label"`
	Name         string              `hcl:"name,label"`
	Id           *string             `hcl:"id"`
	Count        *string             `hcl:"count"`
	SpaceId      *string             `hcl:"space_id"`
	ResourceName string              `hcl:"name"`
	Description  *string             `hcl:"description"`
	Template     []TerraformTemplate `hcl:"template,block"`
}
