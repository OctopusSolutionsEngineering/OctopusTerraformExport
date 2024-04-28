package terraform

type TerraformProjectFeedTrigger struct {
	Type         string                               `hcl:"type,label"`
	Name         string                               `hcl:"name,label"`
	Count        *string                              `hcl:"count"`
	Id           *string                              `hcl:"id"`
	SpaceId      *string                              `hcl:"space_id"`
	ProjectId    string                               `hcl:"project_id"`
	ResourceName string                               `hcl:"name"`
	ChannelId    *string                              `hcl:"channel_id"`
	IsDisabled   *bool                                `hcl:"is_disabled"`
	Package      []TerraformProjectFeedTriggerPackage `hcl:"package,block"`
}

type TerraformProjectFeedTriggerPackage struct {
	DeploymentAction string `hcl:"deployment_action"`
	PackageReference string `hcl:"package_reference"`
}
