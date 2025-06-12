package terraform

type TerraformMachineProxy struct {
	Type         string  `hcl:"type,label"`
	Name         string  `hcl:"name,label"`
	Count        *string `hcl:"count"`
	ResourceName string  `hcl:"name"`
	Id           *string `hcl:"id"`
	SpaceId      *string `hcl:"space_id"`
	Host         string  `hcl:"host"`
	Password     string  `hcl:"password"`
	Username     string  `hcl:"username"`
	Port         *int    `hcl:"port"`
}
