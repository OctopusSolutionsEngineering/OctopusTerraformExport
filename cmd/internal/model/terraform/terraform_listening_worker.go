package terraform

type TerraformListeningWorker struct {
	Type            string   `hcl:"type,label"`
	Name            string   `hcl:"name,label"`
	Count           *string  `hcl:"count"`
	Id              *string  `hcl:"id"`
	SpaceId         *string  `hcl:"space_id"`
	ResourceName    string   `hcl:"name"`
	WorkerPoolIds   []string `hcl:"worker_pool_ids"`
	MachinePolicyId *string  `hcl:"machine_policy_id"`
	Thumbprint      string   `hcl:"thumbprint"`
	Uri             string   `hcl:"uri"`
	ProxyId         *string  `hcl:"proxy_id"`
	IsDisabled      *bool    `hcl:"is_disabled"`
}
