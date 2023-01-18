package singleconverter

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

// SingleProjectConverter exports a single project and its dependencies
type SingleProjectConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
}

// ToHclById is a "smart" export that exports a single project and the dependencies
// required to support it.
func (c SingleProjectConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	project := octopus.Project{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &project)

	if err != nil {
		return err
	}

	thisResource := ResourceDetails{}

	projectName := "project_" + util.SanitizeName(project.Name)

	// The project group is a dependency that we need to lookup
	projectGroupDependencies, err := SingleProjectGroupConverter{
		Client:            c.Client,
		SpaceResourceName: c.SpaceResourceName,
	}.ToHclById(project.ProjectGroupId, false)
	dependencies.AddResource(projectGroupDependencies...)

	// The templates are dependencies that we export as part of the project
	projectTemplates, projectTemplateMap := c.convertTemplates(project.Templates, projectName)
	dependencies.AddResource(projectTemplateMap...)

	// The library variables are dependencies
	for _, v := range project.IncludedLibraryVariableSetIds {
		err := SingleLibraryVariableSetConverter{
			Client:            c.Client,
			SpaceResourceName: c.SpaceResourceName,
		}.ToHclById(v, dependencies)

		if err != nil {
			return err
		}
	}

	// TODO: Need to export deployment process

	thisResource.FileName = "space_population/project_" + projectName + ".tf"
	thisResource.Id = project.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_project." + projectName + ".id}"
	thisResource.ToHcl = func(resources map[string]ResourceDetails) (string, error) {

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
			LifecycleId:                     resources["Lifecycles"+project.LifecycleId].Lookup,
			ProjectGroupId:                  resources["ProjectGroups"+project.ProjectGroupId].Lookup,
			TenantedDeploymentParticipation: project.TenantedDeploymentMode,
			Template:                        projectTemplates,
			IncludedLibraryVariableSets:     c.convertLibraryVariableSets(project.IncludedLibraryVariableSetIds, resources),
			ConnectivityPolicy: terraform.TerraformConnectivityPolicy{
				AllowDeploymentsToNoTargets: project.ProjectConnectivityPolicy.AllowDeploymentsToNoTargets,
				ExcludeUnhealthyTargets:     project.ProjectConnectivityPolicy.ExcludeUnhealthyTargets,
				SkipMachineBehavior:         project.ProjectConnectivityPolicy.SkipMachineBehavior,
			},
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)

	return nil
}

func (c SingleProjectConverter) GetResourceType() string {
	return "Projects"
}

func (c SingleProjectConverter) convertTemplates(actionPackages []octopus.Template, projectName string) ([]terraform.TerraformTemplate, []ResourceDetails) {
	templateMap := make([]ResourceDetails, 0)
	collection := make([]terraform.TerraformTemplate, 0)
	for i, v := range actionPackages {
		collection = append(collection, terraform.TerraformTemplate{
			Name:            v.Name,
			Label:           v.Label,
			HelpText:        v.HelpText,
			DefaultValue:    v.DefaultValue,
			DisplaySettings: v.DisplaySettings,
		})

		templateMap = append(templateMap, ResourceDetails{
			Id:           "",
			ResourceType: "",
			Lookup:       "${octopusdeploy_project." + projectName + ".template[" + fmt.Sprint(i) + "].id}",
			FileName:     "",
			ToHcl:        nil,
		})
	}
	return collection, templateMap
}

func (c SingleProjectConverter) convertLibraryVariableSets(setIds []string, resources map[string]ResourceDetails) []string {
	collection := make([]string, 0)
	for _, v := range setIds {
		collection = append(collection, resources["LibraryVariableSets"+v].Lookup)
	}
	return collection
}
