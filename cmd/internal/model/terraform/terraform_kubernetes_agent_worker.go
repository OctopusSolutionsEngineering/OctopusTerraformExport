package terraform

type TerraformKubernetesAgentTarget struct {
	Type              string   `hcl:"type,label"`
	Name              string   `hcl:"name,label"`
	Count             *string  `hcl:"count"`
	Id                *string  `hcl:"id"`
	SpaceId           *string  `hcl:"space_id"`
	ResourceName      string   `hcl:"name"`
	Thumbprint        string   `hcl:"thumbprint"`
	Uri               string   `hcl:"uri"`
	WorkerPoolIds     []string `hcl:"worker_pool_ids"`
	CommunicationMode *string  `hcl:"communication_mode"`
	IsDisabled        *bool    `hcl:"is_disabled"`
	MachinePolicyId   *string  `hcl:"machine_policy_id"`
	UpgradeLocked     *bool    `hcl:"upgrade_locked"`
}
