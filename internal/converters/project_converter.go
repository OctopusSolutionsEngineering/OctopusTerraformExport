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
	LibraryVariableSetMap    map[string]string
}

func (c ProjectConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.Project]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	channelDependencies := make([]string, 0)

	for _, project := range collection.Items {
		projectName := "project_" + util.SanitizeName(project.Name)
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
			IncludedLibraryVariableSets:     c.convertLibraryVariableSets(project.IncludedLibraryVariableSetIds, c.LibraryVariableSetMap),
			ConnectivityPolicy: terraform.TerraformConnectivityPolicy{
				AllowDeploymentsToNoTargets: project.ProjectConnectivityPolicy.AllowDeploymentsToNoTargets,
				ExcludeUnhealthyTargets:     project.ProjectConnectivityPolicy.ExcludeUnhealthyTargets,
				SkipMachineBehavior:         project.ProjectConnectivityPolicy.SkipMachineBehavior,
			},
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		resultsMap[project.Id] = "${octopusdeploy_project." + projectName + ".id}"
		results["space_population/project_"+projectName+".tf"] = string(file.Bytes())

		// note the project as a channel dependency
		channelDependencies = append(channelDependencies, "octopusdeploy_project."+projectName)

		if project.DeploymentProcessId != nil {
			deploymentProcess, deploymentProcessId, err := DeploymentProcessConverter{
				Client:        c.Client,
				FeedMap:       c.FeedMap,
				WorkPoolMap:   c.WorkPoolMap,
				AccountsMap:   c.AccountsMap,
				ProjectLookup: "${octopusdeploy_project." + projectName + ".id}",
			}.ToHclById(*project.DeploymentProcessId)

			if err != nil {
				return nil, nil, err
			}

			// merge the maps
			for k, v := range deploymentProcess {
				results[k] = v
			}

			// note the deployment project as a channel dependency
			channelDependencies = append(channelDependencies, "octopusdeploy_deployment_process."+deploymentProcessId)
		}

		if project.VariableSetId != nil {
			variableSet, err := VariableSetConverter{
				Client:      c.Client,
				AccountsMap: c.AccountsMap,
			}.ToHclById(*project.VariableSetId, projectName, "${var.octopusdeploy_project."+projectName+".id}")

			if err != nil {
				return nil, nil, err
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
			ProjectLookup:     "${octopusdeploy_project." + projectName + ".id}",
			LifecycleMap:      c.LifecycleMap,
			DependsOn:         channelDependencies,
		}.ToHcl()

		if err != nil {
			return nil, nil, err
		}

		// merge the maps
		for k, v := range lifecycles {
			results[k] = v
		}
	}

	return results, resultsMap, nil
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
