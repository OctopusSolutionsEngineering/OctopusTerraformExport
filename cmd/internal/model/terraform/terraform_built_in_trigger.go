package terraform

type TerraformBuiltInTrigger struct {
	Type                         string                         `hcl:"type,label"`
	Name                         string                         `hcl:"name,label"`
	Count                        *string                        `hcl:"count"`
	Id                           *string                        `hcl:"id"`
	SpaceId                      *string                        `hcl:"space_id"`
	ChannelId                    string                         `hcl:"channel_id"`
	ProjectId                    string                         `hcl:"project_id"`
	ReleaseCreationPackageStepId *string                        `hcl:"release_creation_package_step_id"`
	ReleaseCreationPackage       TerraformBuiltInTriggerPackage `cty:"release_creation_package"`
}

type TerraformBuiltInTriggerPackage struct {
	DeploymentAction string `hcl:"deployment_action"`
	PackageReference string `hcl:"package_reference"`
}
