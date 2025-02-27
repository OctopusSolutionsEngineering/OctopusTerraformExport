package terraform

type TerraformGenericOicdAccount struct {
	Type                            string   `hcl:"type,label"`
	Name                            string   `hcl:"name,label"`
	Id                              *string  `hcl:"id"`
	Count                           *string  `hcl:"count"`
	SpaceId                         *string  `hcl:"space_id"`
	ResourceName                    string   `hcl:"name"`
	Description                     *string  `hcl:"description"`
	Environments                    []string `hcl:"environments"`
	TenantTags                      []string `hcl:"tenant_tags"`
	Tenants                         []string `hcl:"tenants"`
	ExecutionSubjectKeys            []string `hcl:"execution_subject_keys"`
	TenantedDeploymentParticipation *string  `hcl:"tenanted_deployment_participation"`
}
