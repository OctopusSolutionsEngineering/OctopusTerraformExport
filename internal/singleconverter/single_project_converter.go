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
	Client client.OctopusClient
}

func (c SingleProjectConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	project := octopus.Project{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &project)

	if err != nil {
		return err
	}

	return c.toHcl(project, dependencies)
}

func (c SingleProjectConverter) toHcl(project octopus.Project, dependencies *ResourceDetailsCollection) error {
	thisResource := ResourceDetails{}

	projectName := "project_" + util.SanitizeName(project.Name)

	// Export the project group
	err := SingleProjectGroupConverter{
		Client: c.Client,
	}.ToHclById(project.ProjectGroupId, false, dependencies)

	if err != nil {
		return err
	}

	// Export the deployment process
	if project.DeploymentProcessId != nil {
		err = SingleDeploymentProcessConverter{
			Client: c.Client,
		}.ToHclById(*project.DeploymentProcessId, projectName, dependencies)

		if err != nil {
			return err
		}
	}

	// The templates are dependencies that we export as part of the project
	projectTemplates, projectTemplateMap := c.convertTemplates(project.Templates, projectName)
	dependencies.AddResource(projectTemplateMap...)

	// Export the variable set
	if project.VariableSetId != nil {
		err = SingleVariableSetConverter{
			Client: c.Client,
		}.ToHclById(*project.VariableSetId, project.Name, "${octopusdeploy_project."+projectName+".id}", dependencies)

		if err != nil {
			return err
		}
	}

	// Export the library sets
	for _, v := range project.IncludedLibraryVariableSetIds {
		err := SingleLibraryVariableSetConverter{
			Client: c.Client,
		}.ToHclById(v, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the lifecycles
	err = SingleLifecycleConverter{
		Client: c.Client,
	}.ToHclById(project.LifecycleId, dependencies)

	if err != nil {
		return err
	}

	// Export the channels
	err = SingleChannelConverter{
		Client: c.Client,
	}.ToHcl(project.Id, dependencies)

	if err != nil {
		return err
	}

	// Export the triggers
	err = SingleProjectTriggerConverter{
		Client: c.Client,
	}.ToHcl(project.Id, project.Name, dependencies)

	if err != nil {
		return err
	}

	// Export the tenants
	err = SingleTenantConverter{
		Client: c.Client,
	}.ToHcl(project.Id, dependencies)

	if err != nil {
		return err
	}

	// TODO: Need to export git credentials

	thisResource.FileName = "space_population/project_" + projectName + ".tf"
	thisResource.Id = project.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_project." + projectName + ".id}"
	thisResource.ToHcl = func() (string, error) {

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
			LifecycleId:                     dependencies.GetResource("Lifecycles", project.LifecycleId),
			ProjectGroupId:                  dependencies.GetResource("ProjectGroups", project.ProjectGroupId),
			TenantedDeploymentParticipation: project.TenantedDeploymentMode,
			Template:                        projectTemplates,
			IncludedLibraryVariableSets:     c.convertLibraryVariableSets(project.IncludedLibraryVariableSetIds, dependencies),
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

func (c SingleProjectConverter) convertLibraryVariableSets(setIds []string, dependencies *ResourceDetailsCollection) []string {
	collection := make([]string, 0)
	for _, v := range setIds {
		collection = append(collection, dependencies.GetResource("LibraryVariableSets", v))
	}
	return collection
}
