package terraform

type TerraformWorkerPool struct {
	Type         string  `hcl:"type,label"`
	Name         string  `hcl:"name,label"`
	ResourceName string  `hcl:"name"`
	Description  *string `hcl:"description"`
	IsDefault    bool    `hcl:"is_default"`
	SortOrder    int     `hcl:"sort_order"`
	WorkerType   *string `hcl:"worker_type"`
}
