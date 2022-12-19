package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

const terraformFile = internal.PopulateSpaceDir + "/projects.tf"

type ProjectConverter struct {
	Client client.OctopusClient
}

func (c ProjectConverter) ToHcl() (map[string]string, error) {
	collection := model.GeneralCollection[model.Project]{}
	err := c.Client.GetAllResources("Projects", &collection)

	if err != nil {
		return nil, err
	}

	output := ""

	for _, project := range collection.Items {
		terraformResource := model.TerraformProject{
			Type:                            "octopusdeploy_project",
			Name:                            "octopus_project_" + util.SanitizeName(project.Name),
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

	return map[string]string{
		terraformFile: output,
	}, nil
}

func (c ProjectConverter) ToHclById(id string) (map[string]string, error) {
	resource := model.Project{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, err
	}

	terraformResource := model.TerraformProject{
		Type:                            "octopusdeploy_project",
		Name:                            "octopus_space_" + util.SanitizeName(resource.Name),
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

	return map[string]string{
		terraformFile: string(file.Bytes()),
	}, nil
}

func (c ProjectConverter) ToHclByName(name string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (c ProjectConverter) GetResourceType() string {
	return "Projects"
}
