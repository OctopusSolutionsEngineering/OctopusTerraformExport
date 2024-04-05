package terraform

type TerraformSshConnectionDeploymentTarget struct {
	Id                 *string  `hcl:"id"`
	SpaceId            *string  `hcl:"space_id"`
	Type               string   `hcl:"type,label"`
	Name               string   `hcl:"name,label"`
	Count              *string  `hcl:"count"`
	AccountId          string   `hcl:"account_id"`
	Environments       []string `hcl:"environments"`
	Fingerprint        string   `hcl:"fingerprint"`
	Host               string   `hcl:"host"`
	ResourceName       string   `hcl:"name"`
	Roles              []string `hcl:"roles"`
	DotNetCorePlatform *string  `hcl:"dot_net_core_platform"`
	MachinePolicyId    *string  `hcl:"machine_policy_id"`
	TenantTags         []string `hcl:"tenant_tags"`
}
