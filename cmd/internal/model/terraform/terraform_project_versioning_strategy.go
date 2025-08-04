package terraform

type TerraformProjectVersioningStrategy struct {
	Type               string                                          `hcl:"type,label"`
	Name               string                                          `hcl:"name,label"`
	Count              *string                                         `hcl:"count"`
	ProjectId          string                                          `hcl:"project_id"`
	SpaceId            *string                                         `hcl:"space_id"`
	DonorPackage       *TerraformProjectVersioningStrategyDonorPackage `hcl:"donor_package,block"`
	DonorPackageStepId *string                                         `hcl:"donor_package_step_id"`
	Template           *string                                         `hcl:"template"`
}

type TerraformProjectVersioningStrategyDonorPackage struct {
	DeploymentAction string `hcl:"deployment_action"`
	PackageReference string `hcl:"package_reference"`
}
