package terraform

// TerraformProcessStepsOrder represents the order of child steps in a parent step/
type TerraformProcessChildStepsOrder struct {
	Type      string   `hcl:"type,label"`
	Name      string   `hcl:"name,label"`
	Count     *string  `hcl:"count"`
	Id        *string  `hcl:"id"`
	ProcessId string   `hcl:"process_id"`
	ParentId  string   `hcl:"parent_id"`
	Children  []string `hcl:"children"`
}
