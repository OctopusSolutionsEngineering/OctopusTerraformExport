package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type ProjectConverter struct {
	Client                   client.OctopusClient
	SpaceResourceName        string
	ProjectGroupResourceName string
	ProjectGroupId           string
	FeedMap                  map[string]string
	LifecycleMap             map[string]string
	WorkPoolMap              map[string]string
}

func (c ProjectConverter) ToHcl() (map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.Project]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, err
	}

	results := map[string]string{}

	for _, project := range collection.Items {
		projectName := util.SanitizeName(project.Slug)
		terraformResource := terraform.TerraformProject{
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
			LifecycleId:                     c.LifecycleMap[project.LifecycleId],
			ProjectGroupId:                  "${octopusdeploy_project_group." + c.ProjectGroupResourceName + ".id}",
			TenantedDeploymentParticipation: project.TenantedDeploymentMode,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		results[projectName+".tf"] = string(file.Bytes())

		if project.DeploymentProcessId != nil {
			deploymentProcess, err := DeploymentProcessConverter{
				Client:      c.Client,
				FeedMap:     c.FeedMap,
				WorkPoolMap: c.WorkPoolMap,
			}.ToHclById(*project.DeploymentProcessId, projectName)

			if err != nil {
				return nil, err
			}

			// merge the maps
			for k, v := range deploymentProcess {
				results[k] = v
			}
		}

		if project.VariableSetId != nil {
			variableSet, err := VariableSetConverter{Client: c.Client}.ToHclById(*project.VariableSetId, projectName)

			if err != nil {
				return nil, err
			}

			// merge the maps
			for k, v := range variableSet {
				results[k] = v
			}
		}

	}

	return results, nil
}

func (c ProjectConverter) ToHclById(id string) (map[string]string, error) {
	resource := octopus.Project{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, err
	}

	resourceName := util.SanitizeName(resource.Name)

	terraformResource := terraform.TerraformProject{
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
		ProjectGroupId:                  "${octopusdeploy_project_group." + c.ProjectGroupResourceName + ".id}",
		TenantedDeploymentParticipation: resource.TenantedDeploymentMode,
	}
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

	return map[string]string{
		resourceName + ".tf": string(file.Bytes()),
	}, nil
}

func (c ProjectConverter) ToHclByName(name string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (c ProjectConverter) GetResourceType() string {
	return "ProjectGroups/" + c.ProjectGroupId + "/projects"
}
