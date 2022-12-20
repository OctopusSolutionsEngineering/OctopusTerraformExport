package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type ProjectConverter struct {
	Client client.OctopusClient
}

func (c ProjectConverter) ToHcl() (map[string]string, error) {
	collection := model.GeneralCollection[model.Project]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, err
	}

	results := map[string]string{}

	for _, project := range collection.Items {
		projectName := util.SanitizeName(project.Id)
		terraformResource := model.TerraformProject{
			Type:                            "octopusdeploy_project",
			Name:                            projectName,
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

		results[internal.PopulateSpaceDir+"/"+projectName+".tf"] = string(file.Bytes())

		deploymentProcess, err := DeploymentProcessConverter{Client: c.Client}.ToHclById(project.DeploymentProcessId, projectName)

		if err != nil {
			return nil, err
		}

		// merge the maps
		for k, v := range deploymentProcess {
			results[k] = v
		}

	}

	return results, nil
}

func (c ProjectConverter) ToHclById(id string) (map[string]string, error) {
	resource := model.Project{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, err
	}

	resourceName := util.SanitizeName(resource.Name)

	terraformResource := model.TerraformProject{
		Type:                            "octopusdeploy_project",
		Name:                            resourceName,
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
		internal.PopulateSpaceDir + "/" + resourceName + ".tf": string(file.Bytes()),
	}, nil
}

func (c ProjectConverter) ToHclByName(name string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (c ProjectConverter) GetResourceType() string {
	return "Projects"
}
