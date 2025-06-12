package terraform

// TerraformProcessStepsOrder represents the order of steps in a process.
type TerraformProcessStepsOrder struct {
	Type      string   `hcl:"type,label"`
	Name      string   `hcl:"name,label"`
	Count     *string  `hcl:"count"`
	Id        *string  `hcl:"id"`
	ProcessId string   `hcl:"process_id"`
	Steps     []string `hcl:"steps"`
}
