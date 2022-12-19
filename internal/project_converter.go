package internal

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
	"regexp"
	"strings"
)

type projectTerraform struct {
	Type                            string `hcl:"type,label"`
	Name                            string `hcl:"name,label"`
	ResourceName                    string `hcl:"name"`
	AutoCreateRelease               bool   `hcl:"auto_create_release"`
	DefaultGuidedFailureMode        string `hcl:"default_guided_failure_mode"`
	DefaultToSkipIfAlreadyInstalled bool   `hcl:"default_to_skip_if_already_installed"`
	Description                     string `hcl:"description"`
	DiscreteChannelRelease          bool   `hcl:"discrete_channel_release"`
	IsDisabled                      bool   `hcl:"is_disabled"`
	IsVersionControlled             bool   `hcl:"is_version_controlled"`
	LifecycleId                     string `hcl:"lifecycle_id"`
	ProjectGroupId                  string `hcl:"project_group_id"`
	TenantedDeploymentParticipation string `hcl:"tenanted_deployment_participation"`
}

type ProjectConverter struct {
	Client OctopusClient
}

func (c ProjectConverter) ToHcl() (string, error) {
	collection := model.GeneralCollection[model.Project]{}
	err := c.Client.GetAllResources("Projects", &collection)

	if err != nil {
		return "", err
	}

	allowedChars := regexp.MustCompile(`[^A-Za-z0-9]`)

	output := ""

	for _, project := range collection.Items {
		terraformResource := projectTerraform{
			Type:                            "octopusdeploy_project",
			Name:                            "octopus_project_" + allowedChars.ReplaceAllString(strings.ToLower(project.Name), "_"),
			ResourceName:                    project.Name,
			AutoCreateRelease:               project.AutoCreateRelease,
			DefaultGuidedFailureMode:        project.DefaultGuidedFailureMode,
			DefaultToSkipIfAlreadyInstalled: project.DefaultToSkipIfAlreadyInstalled,
			Description:                     project.Description,
			DiscreteChannelRelease:          project.DiscreteChannelRelease,
			IsDisabled:                      project.IsDisabled,
			IsVersionControlled:             project.IsVersionControlled,
			LifecycleId:                     project.LifecycleId,
			ProjectGroupId:                  project.ProjectGroupId,
			TenantedDeploymentParticipation: project.TenantedDeploymentMode,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		output += string(file.Bytes())
	}

	return output, nil
}

func (c ProjectConverter) ToHclById(id string) (string, error) {
	resource := model.Project{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return "", err
	}

	allowedChars := regexp.MustCompile(`[^A-Za-z0-9]`)

	terraformResource := projectTerraform{
		Type:                            "octopusdeploy_project",
		Name:                            "octopus_space_" + allowedChars.ReplaceAllString(strings.ToLower(resource.Name), "_"),
		ResourceName:                    resource.Name,
		AutoCreateRelease:               resource.AutoCreateRelease,
		DefaultGuidedFailureMode:        resource.DefaultGuidedFailureMode,
		DefaultToSkipIfAlreadyInstalled: resource.DefaultToSkipIfAlreadyInstalled,
		Description:                     resource.Description,
		DiscreteChannelRelease:          resource.DiscreteChannelRelease,
		IsDisabled:                      resource.IsDisabled,
		IsVersionControlled:             resource.IsVersionControlled,
		LifecycleId:                     resource.LifecycleId,
		ProjectGroupId:                  resource.ProjectGroupId,
		TenantedDeploymentParticipation: resource.TenantedDeploymentMode,
	}
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
	return string(file.Bytes()), nil
}

func (c ProjectConverter) ToHclByName(name string) (string, error) {
	return "", nil
}

func (c ProjectConverter) GetResourceType() string {
	return "Projects"
}
