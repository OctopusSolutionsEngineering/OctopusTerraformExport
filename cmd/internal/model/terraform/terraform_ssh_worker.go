package terraform

type TerraformSshWorker struct {
	Type            string   `hcl:"type,label"`
	Name            string   `hcl:"name,label"`
	Count           *string  `hcl:"count"`
	Id              *string  `hcl:"id"`
	SpaceId         *string  `hcl:"space_id"`
	ResourceName    string   `hcl:"name"`
	AccountId       string   `hcl:"account_id"`
	DotnetPlatform  string   `hcl:"dotnet_platform"`
	Fingerprint     string   `hcl:"fingerprint"`
	Host            string   `hcl:"host"`
	Port            int      `hcl:"port"`
	WorkerPoolIds   []string `hcl:"worker_pool_ids"`
	MachinePolicyId *string  `hcl:"machine_policy_id"`
	Uri             string   `hcl:"uri"`
	ProxyId         *string  `hcl:"proxy_id"`
	IsDisabled      *bool    `hcl:"is_disabled"`
}
