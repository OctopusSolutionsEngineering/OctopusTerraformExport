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
	Client                client.OctopusClient
	SpaceResourceName     string
	FeedMap               map[string]string
	LifecycleMap          map[string]string
	WorkPoolMap           map[string]string
	AccountsMap           map[string]string
	LibraryVariableSetMap map[string]string
}

func (c ProjectGroupConverter) ToHcl() (map[string]string, map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.ProjectGroup]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}
	templatesMap := map[string]string{}

	for _, project := range collection.Items {
		projectName := "project_group_" + util.SanitizeNamePointer(project.Name)

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

			results["space_population/project_"+projectName+".tf"] = string(file.Bytes())
		}

		// Convert the projects
		projects, projectsMap, projectTemplatesMap, err := ProjectConverter{
			Client:                   c.Client,
			SpaceResourceName:        c.SpaceResourceName,
			ProjectGroupResourceName: projectName,
			ProjectGroupId:           project.Id,
			FeedMap:                  c.FeedMap,
			LifecycleMap:             c.LifecycleMap,
			WorkPoolMap:              c.WorkPoolMap,
			LibraryVariableSetMap:    c.LibraryVariableSetMap,
		}.ToHcl()
		if err != nil {
			return nil, nil, nil, err
		}

		// merge the maps
		for k, v := range projects {
			results[k] = v
		}

		for k, v := range projectsMap {
			resultsMap[k] = v
		}

		for k, v := range projectTemplatesMap {
			templatesMap[k] = v
		}
	}

	return results, resultsMap, templatesMap, nil
}

func (c ProjectGroupConverter) ToHclByName(name string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (c ProjectGroupConverter) GetResourceType() string {
	return "ProjectGroups"
}
