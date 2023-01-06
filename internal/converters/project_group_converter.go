package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type ProjectGroupConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	FeedMap           map[string]string
	LifecycleMap      map[string]string
	WorkPoolMap       map[string]string
	AccountsMap       map[string]string
}

func (c ProjectGroupConverter) ToHcl() (map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.ProjectGroup]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, err
	}

	results := map[string]string{}

	for _, project := range collection.Items {
		projectName := "project_group_" + util.SanitizeNamePointer(project.Name)
		var projectGroupVar string

		if *project.Name == "Default Project Group" {
			// todo - create lookup for existing project group
		} else {
			terraformResource := terraform.TerraformProjectGroup{
				Type:         "octopusdeploy_project_group",
				Name:         projectName,
				ResourceName: project.Name,
				Description:  project.Description,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			results["space_population/"+projectName+".tf"] = string(file.Bytes())
			projectGroupVar = "${octopusdeploy_project_group." + projectName + ".id}"
		}

		// Convert the projects
		projects, err := ProjectConverter{
			Client:                   c.Client,
			SpaceResourceName:        c.SpaceResourceName,
			ProjectGroupResourceName: projectName,
			ProjectGroupId:           projectGroupVar,
			FeedMap:                  c.FeedMap,
			LifecycleMap:             c.LifecycleMap,
			WorkPoolMap:              c.WorkPoolMap,
		}.ToHcl()
		if err != nil {
			return nil, err
		}

		// merge the maps
		for k, v := range projects {
			results[k] = v
		}
	}

	return results, nil
}

func (c ProjectGroupConverter) ToHclById(id string) (map[string]string, error) {
	resource := octopus.Project{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, err
	}

	results := map[string]string{}

	resourceName := util.SanitizeNamePointer(resource.Name)

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
		ProjectGroupId:                  resource.ProjectGroupId,
		TenantedDeploymentParticipation: resource.TenantedDeploymentMode,
	}
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
	results["space_population/"+resourceName+".tf"] = string(file.Bytes())

	// Convert the projects
	projects, err := ProjectConverter{
		Client:                   c.Client,
		SpaceResourceName:        c.SpaceResourceName,
		ProjectGroupResourceName: resourceName,
		ProjectGroupId:           resource.Id,
		AccountsMap:              c.AccountsMap,
	}.ToHcl()
	if err != nil {
		return nil, err
	}

	// merge the maps
	for k, v := range projects {
		results[k] = v
	}

	return map[string]string{
		resourceName + ".tf": string(file.Bytes()),
	}, nil
}

func (c ProjectGroupConverter) ToHclByName(name string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (c ProjectGroupConverter) GetResourceType() string {
	return "ProjectGroups"
}
