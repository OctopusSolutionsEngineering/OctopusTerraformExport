package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type ProjectConverter struct {
	Client client.OctopusClient
}

func (c ProjectConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Project]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c ProjectConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	project := octopus.Project{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &project)

	if err != nil {
		return err
	}

	return c.toHcl(project, true, dependencies)
}

func (c ProjectConverter) toHcl(project octopus.Project, recursive bool, dependencies *ResourceDetailsCollection) error {
	thisResource := ResourceDetails{}

	projectName := "project_" + util.SanitizeName(project.Name)

	if recursive {
		err := c.exportDependencies(project, projectName, dependencies)

		if err != nil {
			return err
		}
	}

	err := c.exportChildDependencies(project, recursive, projectName, dependencies)

	if err != nil {
		return err
	}

	// The templates are dependencies that we export as part of the project
	projectTemplates, projectTemplateMap := c.convertTemplates(project.Templates, projectName)
	dependencies.AddResource(projectTemplateMap...)

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
			IncludedLibraryVariableSets:     c.convertLibraryVariableSets(project.IncludedLibraryVariableSetIds, dependencies),
			TenantedDeploymentParticipation: project.TenantedDeploymentMode,
			Template:                        projectTemplates,
			ConnectivityPolicy: terraform.TerraformConnectivityPolicy{
				AllowDeploymentsToNoTargets: project.ProjectConnectivityPolicy.AllowDeploymentsToNoTargets,
				ExcludeUnhealthyTargets:     project.ProjectConnectivityPolicy.ExcludeUnhealthyTargets,
				SkipMachineBehavior:         project.ProjectConnectivityPolicy.SkipMachineBehavior,
			},
			GitLibraryPersistenceSettings:          c.convertLibraryGitPersistence(project, projectName, dependencies),
			GitAnonymousPersistenceSettings:        c.convertAnonymousGitPersistence(project, projectName),
			GitUsernamePasswordPersistenceSettings: c.convertUsernamePasswordGitPersistence(project, projectName),
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		if terraformResource.GitUsernamePasswordPersistenceSettings != nil {
			secretVariableResource := terraform.TerraformVariable{
				Name:        projectName + "_git_password",
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "The git password for the project \"" + project.Name + "\"",
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		if terraformResource.GitUsernamePasswordPersistenceSettings != nil ||
			terraformResource.GitAnonymousPersistenceSettings != nil ||
			terraformResource.GitLibraryPersistenceSettings != nil {
			secretVariableResource := terraform.TerraformVariable{
				Name:        projectName + "_git_base_path",
				Type:        "string",
				Nullable:    false,
				Sensitive:   false,
				Description: "The git base path for \"" + project.Name + "\"",
				Default:     &project.PersistenceSettings.BasePath,
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)

	return nil
}

func (c ProjectConverter) GetResourceType() string {
	return "Projects"
}

func (c ProjectConverter) convertTemplates(actionPackages []octopus.Template, projectName string) ([]terraform.TerraformTemplate, []ResourceDetails) {
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
			Id:           v.Id,
			ResourceType: "ProjectTemplates",
			Lookup:       "${octopusdeploy_project." + projectName + ".template[" + fmt.Sprint(i) + "].id}",
			FileName:     "",
			ToHcl:        nil,
		})
	}
	return collection, templateMap
}

func (c ProjectConverter) convertLibraryVariableSets(setIds []string, dependencies *ResourceDetailsCollection) []string {
	collection := make([]string, 0)
	for _, v := range setIds {
		collection = append(collection, dependencies.GetResource("LibraryVariableSets", v))
	}
	return collection
}

func (c ProjectConverter) convertLibraryGitPersistence(project octopus.Project, projectName string, dependencies *ResourceDetailsCollection) *terraform.TerraformGitLibraryPersistenceSettings {
	if project.PersistenceSettings.Credentials.Type != "Reference" {
		return nil
	}

	return &terraform.TerraformGitLibraryPersistenceSettings{
		GitCredentialId:   dependencies.GetResource("Git-Credentials", project.PersistenceSettings.Credentials.Id),
		Url:               project.PersistenceSettings.Url,
		BasePath:          "${var." + projectName + "_git_base_path}",
		DefaultBranch:     project.PersistenceSettings.DefaultBranch,
		ProtectedBranches: project.PersistenceSettings.ProtectedBranchNamePatterns,
	}
}

func (c ProjectConverter) convertAnonymousGitPersistence(project octopus.Project, projectName string) *terraform.TerraformGitAnonymousPersistenceSettings {
	if project.PersistenceSettings.Credentials.Type != "Anonymous" {
		return nil
	}

	return &terraform.TerraformGitAnonymousPersistenceSettings{
		Url:               project.PersistenceSettings.Url,
		BasePath:          "${var." + projectName + "_git_base_path}",
		DefaultBranch:     project.PersistenceSettings.DefaultBranch,
		ProtectedBranches: project.PersistenceSettings.ProtectedBranchNamePatterns,
	}
}

func (c ProjectConverter) convertUsernamePasswordGitPersistence(project octopus.Project, projectName string) *terraform.TerraformGitUsernamePasswordPersistenceSettings {
	if project.PersistenceSettings.Credentials.Type != "UsernamePassword" {
		return nil
	}

	return &terraform.TerraformGitUsernamePasswordPersistenceSettings{
		Url:               project.PersistenceSettings.Url,
		Username:          project.PersistenceSettings.Credentials.Username,
		Password:          "${var." + projectName + "_git_password}",
		BasePath:          "${var." + projectName + "_git_base_path}",
		DefaultBranch:     project.PersistenceSettings.DefaultBranch,
		ProtectedBranches: project.PersistenceSettings.ProtectedBranchNamePatterns,
	}
}

// exportChildDependencies exports those dependencies that are always required regardless of the recursive flag.
// These are resources that do not expose an API for bulk retrieval, or those whose resource names benefit
// from the parent's name (i.e. a deployment process resource name will be "deployment_process_<projectname>").
func (c ProjectConverter) exportChildDependencies(project octopus.Project, recursive bool, projectName string, dependencies *ResourceDetailsCollection) error {
	err := ChannelConverter{
		Client: c.Client,
		DependsOn: map[string]string{
			"DeploymentProcesses": util.EmptyIfNil(project.DeploymentProcessId),
		},
	}.ToHclByProjectId(project.Id, recursive, dependencies)

	if err != nil {
		return err
	}

	// Export the deployment process
	if project.DeploymentProcessId != nil {
		err = DeploymentProcessConverter{
			Client: c.Client,
		}.ToHclById(*project.DeploymentProcessId, true, projectName, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the variable set
	if project.VariableSetId != nil {
		err = VariableSetConverter{
			Client: c.Client,
		}.ToHclById(*project.VariableSetId, true, project.Name, "${octopusdeploy_project."+projectName+".id}", dependencies)

		if err != nil {
			return err
		}
	}

	// Export the triggers
	err = ProjectTriggerConverter{
		Client: c.Client,
	}.ToHclByProjectId(project.Id, project.Name, dependencies)

	if err != nil {
		return err
	}

	return nil
}

func (c ProjectConverter) exportDependencies(project octopus.Project, projectName string, dependencies *ResourceDetailsCollection) error {
	// Export the project group
	err := ProjectGroupConverter{
		Client: c.Client,
	}.ToHclById(project.ProjectGroupId, false, dependencies)

	if err != nil {
		return err
	}

	// Export the library sets
	for _, v := range project.IncludedLibraryVariableSetIds {
		err := LibraryVariableSetConverter{
			Client: c.Client,
		}.ToHclById(v, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the lifecycles
	err = LifecycleConverter{
		Client: c.Client,
	}.ToHclById(project.LifecycleId, dependencies)

	if err != nil {
		return err
	}

	// Export the tenants
	err = TenantConverter{
		Client: c.Client,
	}.ToHclByProjectId(project.Id, dependencies)

	if err != nil {
		return err
	}

	// Export the git credentials
	if project.PersistenceSettings.Credentials.Type == "Reference" {
		err = GitCredentialsConverter{
			Client: c.Client,
		}.ToHclById(project.PersistenceSettings.Credentials.Id, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}
