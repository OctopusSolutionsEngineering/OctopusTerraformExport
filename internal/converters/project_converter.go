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
	AccountsMap              map[string]string
}

func (c ProjectConverter) ToHcl() (map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.Project]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, err
	}

	results := map[string]string{}

	for _, project := range collection.Items {

		/*
			Convert library variable sets.
			This may be duplicated if multiple projects reference the same set,
			but that is fine as converting the set multiple times produces the
			same outcome.
		*/
		variableSetMap := map[string]string{}
		for _, v := range project.IncludedLibraryVariableSetIds {
			variables, variableMap, err := LibraryVariableSetConverter{
				Client:            c.Client,
				SpaceResourceName: c.SpaceResourceName,
			}.ToHclById(v)

			if err != nil {
				return nil, err
			}

			// merge the maps
			for k, v := range variableMap {
				variableSetMap[k] = v
			}

			// merge the results
			for k, v := range variables {
				results[k] = v
			}
		}

		projectName := util.SanitizeNamePointer(project.Name)
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
			Template:                        c.convertTemplates(project.Templates),
			IncludedLibraryVariableSets:     c.convertLibraryVariableSets(project.IncludedLibraryVariableSetIds, variableSetMap),
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		results["space_population/"+projectName+".tf"] = string(file.Bytes())

		if project.DeploymentProcessId != nil {
			deploymentProcess, err := DeploymentProcessConverter{
				Client:      c.Client,
				FeedMap:     c.FeedMap,
				WorkPoolMap: c.WorkPoolMap,
				AccountsMap: c.AccountsMap,
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
			variableSet, err := VariableSetConverter{
				Client:      c.Client,
				AccountsMap: c.AccountsMap,
			}.ToHclById(*project.VariableSetId, projectName)

			if err != nil {
				return nil, err
			}

			// merge the maps
			for k, v := range variableSet {
				results[k] = v
			}
		}

		// export the channels
		lifecycles, _, err := ChannelConverter{
			Client:            c.Client,
			SpaceResourceName: c.SpaceResourceName,
			ProjectId:         project.Id,
			LifecycleMap:      c.LifecycleMap,
		}.ToHcl()

		if err != nil {
			return nil, err
		}

		// merge the maps
		for k, v := range lifecycles {
			results[k] = v
		}
	}

	return results, nil
}

func (c ProjectConverter) GetResourceType() string {
	return "ProjectGroups/" + c.ProjectGroupId + "/projects"
}

func (c ProjectConverter) convertTemplates(actionPackages []octopus.Template) []terraform.TerraformTemplate {
	collection := make([]terraform.TerraformTemplate, 0)
	for _, v := range actionPackages {
		collection = append(collection, terraform.TerraformTemplate{
			Name:            v.Name,
			Label:           v.Label,
			HelpText:        v.HelpText,
			DefaultValue:    v.DefaultValue,
			DisplaySettings: v.DisplaySettings,
		})
	}
	return collection
}

func (c ProjectConverter) convertLibraryVariableSets(setIds []string, libraryMap map[string]string) []string {
	collection := make([]string, 0)
	for _, v := range setIds {
		collection = append(collection, libraryMap[v])
	}
	return collection
}
