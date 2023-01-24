package terraform

type TerraformSshConnectionDeploymentTarget struct {
	Type               string   `hcl:"type,label"`
	Name               string   `hcl:"name,label"`
	AccountId          string   `hcl:"account_id"`
	Environments       []string `hcl:"environments"`
	Fingerprint        string   `hcl:"fingerprint"`
	Host               string   `hcl:"host"`
	ResourceName       string   `hcl:"name"`
	Roles              []string `hcl:"roles"`
	DotNetCorePlatform *string  `hcl:"dot_net_core_platform"`
	MachinePolicyId    *string  `hcl:"machine_policy_id"`
}
