package terraform

// TerraformProcessStepsOrder represents the order of steps in a process. It i used to create
// octopusdeploy_process_steps_order and octopusdeploy_process_child_steps_order.
type TerraformProcessStepsOrder struct {
	Type      string   `hcl:"type,label"`
	Name      string   `hcl:"name,label"`
	Count     *string  `hcl:"count"`
	Id        *string  `hcl:"id"`
	ProcessId string   `hcl:"process_id"`
	Steps     []string `hcl:"steps"`
}
