package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/strutil"
)

type ProjectConverter struct {
	Client                      client.OctopusClient
	LifecycleConverter          ConverterAndLookupById
	GitCredentialsConverter     ConverterAndLookupById
	LibraryVariableSetConverter ConverterAndLookupById
	ProjectGroupConverter       ConverterAndLookupById
	DeploymentProcessConverter  ConverterAndLookupByIdAndName
	TenantConverter             ConverterAndLookupByProjectId
	ProjectTriggerConverter     ConverterByProjectIdWithName
	VariableSetConverter        ConverterAndLookupByIdWithNameAndParent
	ChannelConverter            ConverterAndLookupByProjectIdWithTerraDependencies
	RunbookConverter            ConverterAndLookupByIdAndName
	IgnoreCacManagedValues      bool
	ExcludeAllRunbooks          bool
	IgnoreProjectChanges        bool
}

func (c ProjectConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Project]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, false, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

// ToHclLookupById exports a self-contained representation of the project where external resources like
// environments, lifecycles, feeds, accounts etc are resolved with data lookups.
func (c ProjectConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	project := octopus.Project{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &project)

	if err != nil {
		return err
	}

	return c.toHcl(project, false, true, dependencies)
}

func (c ProjectConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	project := octopus.Project{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &project)

	if err != nil {
		return err
	}

	return c.toHcl(project, true, false, dependencies)
}

func (c ProjectConverter) toHcl(project octopus.Project, recursive bool, lookups bool, dependencies *ResourceDetailsCollection) error {
	thisResource := ResourceDetails{}

	projectName := "project_" + sanitizer.SanitizeName(project.Name)

	if recursive {
		err := c.exportDependencies(project, projectName, dependencies)

		if err != nil {
			return err
		}
	} else if lookups {
		err := c.exportDependencyLookups(project, projectName, dependencies)

		if err != nil {
			return err
		}
	}

	err := c.exportChildDependencies(recursive, lookups, project, projectName, dependencies)

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
			Type:                                   "octopusdeploy_project",
			Name:                                   projectName,
			ResourceName:                           "${var." + projectName + "_name}",
			AutoCreateRelease:                      project.AutoCreateRelease,
			DefaultGuidedFailureMode:               project.DefaultGuidedFailureMode,
			DefaultToSkipIfAlreadyInstalled:        project.DefaultToSkipIfAlreadyInstalled,
			Description:                            project.Description,
			DiscreteChannelRelease:                 project.DiscreteChannelRelease,
			IsDisabled:                             project.IsDisabled,
			IsVersionControlled:                    project.IsVersionControlled,
			LifecycleId:                            dependencies.GetResource("Lifecycles", project.LifecycleId),
			ProjectGroupId:                         dependencies.GetResource("ProjectGroups", project.ProjectGroupId),
			IncludedLibraryVariableSets:            c.convertLibraryVariableSets(project.IncludedLibraryVariableSetIds, dependencies),
			TenantedDeploymentParticipation:        project.TenantedDeploymentMode,
			Template:                               projectTemplates,
			ConnectivityPolicy:                     c.convertConnectivityPolicy(project),
			GitLibraryPersistenceSettings:          c.convertLibraryGitPersistence(project, projectName, dependencies),
			GitAnonymousPersistenceSettings:        c.convertAnonymousGitPersistence(project, projectName),
			GitUsernamePasswordPersistenceSettings: c.convertUsernamePasswordGitPersistence(project, projectName),
			VersioningStrategy:                     c.convertVersioningStrategy(project),
		}

		if c.IgnoreProjectChanges {
			all := "all"
			terraformResource.Lifecycle = &terraform.TerraformLifecycleMetaArgument{
				IgnoreAllChanges: &all,
			}
		}

		file := hclwrite.NewEmptyFile()

		c.writeProjectNameVariable(file, projectName, project.Name)

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
			Type: hclsyntax.TokenComment,
			Bytes: []byte("# Import existing resources with the following commands:\n" +
				"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + project.Name + "\") | .Id')\n" +
				"# terraform import octopusdeploy_project." + projectName + " ${RESOURCE_ID}\n"),
			SpacesBefore: 0,
		}})

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
			hcl.WriteUnquotedAttribute(block, "type", "string")
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
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			gitUrlVariable := terraform.TerraformVariable{
				Name:        projectName + "_git_url",
				Type:        "string",
				Nullable:    false,
				Sensitive:   false,
				Description: "The git url for \"" + project.Name + "\"",
				Default:     &project.PersistenceSettings.Url,
			}

			block = gohcl.EncodeAsBlock(gitUrlVariable, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
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

func (c ProjectConverter) writeProjectNameVariable(file *hclwrite.File, projectName string, projectResourceName string) {
	secretVariableResource := terraform.TerraformVariable{
		Name:        projectName + "_name",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The name of the project exported from " + projectResourceName,
		Default:     &projectResourceName,
	}

	block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)
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

func (c ProjectConverter) convertConnectivityPolicy(project octopus.Project) *terraform.TerraformConnectivityPolicy {
	if c.IgnoreCacManagedValues {
		return nil
	}

	return &terraform.TerraformConnectivityPolicy{
		AllowDeploymentsToNoTargets: project.ProjectConnectivityPolicy.AllowDeploymentsToNoTargets,
		ExcludeUnhealthyTargets:     project.ProjectConnectivityPolicy.ExcludeUnhealthyTargets,
		SkipMachineBehavior:         project.ProjectConnectivityPolicy.SkipMachineBehavior,
	}
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
		Url:               "${var." + projectName + "_git_url}",
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
		Url:               "${var." + projectName + "_git_url}",
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
		Url:               "${var." + projectName + "_git_url}",
		Username:          project.PersistenceSettings.Credentials.Username,
		Password:          "${var." + projectName + "_git_password}",
		BasePath:          "${var." + projectName + "_git_base_path}",
		DefaultBranch:     project.PersistenceSettings.DefaultBranch,
		ProtectedBranches: project.PersistenceSettings.ProtectedBranchNamePatterns,
	}
}

func (c ProjectConverter) convertVersioningStrategy(project octopus.Project) *terraform.TerraformVersioningStrategy {
	if c.IgnoreCacManagedValues {
		return nil
	}

	// Versioning based on packages creates a circular reference that Terraform can not resolve. The project
	// needs to know the name of the step and package to base the versioning on, and the deployment process
	// needs to know the project to attach itself to. If the versioning strategy is set to use packages,
	// simply return nil.
	if strutil.EmptyIfNil(project.VersioningStrategy.DonorPackageStepId) != "" ||
		project.VersioningStrategy.DonorPackage != nil {
		return nil
	}

	versioningStrategy := terraform.TerraformVersioningStrategy{
		Template:           project.VersioningStrategy.Template,
		DonorPackageStepId: nil,
		DonorPackage:       nil,
	}

	if project.VersioningStrategy.DonorPackage != nil {
		versioningStrategy.DonorPackage = &terraform.TerraformDonorPackage{
			DeploymentAction: project.VersioningStrategy.DonorPackage.DeploymentAction,
			PackageReference: project.VersioningStrategy.DonorPackage.PackageReference,
		}
	}

	return &versioningStrategy
}

// exportChildDependencies exports those dependencies that are always required regardless of the recursive flag.
// These are resources that do not expose an API for bulk retrieval, or those whose resource names benefit
// from the parent's name (i.e. a deployment process resource name will be "deployment_process_<projectname>").
func (c ProjectConverter) exportChildDependencies(recursive bool, lookup bool, project octopus.Project, projectName string, dependencies *ResourceDetailsCollection) error {
	var err error
	if lookup {
		err = c.ChannelConverter.ToHclLookupByProjectIdWithTerraDependencies(project.Id, map[string]string{
			"DeploymentProcesses": strutil.EmptyIfNil(project.DeploymentProcessId),
		}, dependencies)
	} else {
		err = c.ChannelConverter.ToHclByProjectIdWithTerraDependencies(project.Id, map[string]string{
			"DeploymentProcesses": strutil.EmptyIfNil(project.DeploymentProcessId),
		}, dependencies)
	}

	if err != nil {
		return err
	}

	// Export the deployment process
	if project.DeploymentProcessId != nil && !c.IgnoreCacManagedValues {
		var err error
		if lookup {
			err = c.DeploymentProcessConverter.ToHclLookupByIdAndName(*project.DeploymentProcessId, projectName, dependencies)
		} else {
			err = c.DeploymentProcessConverter.ToHclByIdAndName(*project.DeploymentProcessId, projectName, dependencies)
		}

		if err != nil {
			return err
		}
	}

	// Export the variable set
	if project.VariableSetId != nil {
		var err error
		if lookup {
			err = c.VariableSetConverter.ToHclLookupByIdAndName(*project.VariableSetId, project.Name, "${octopusdeploy_project."+projectName+".id}", dependencies)
		} else {
			err = c.VariableSetConverter.ToHclByIdAndName(*project.VariableSetId, project.Name, "${octopusdeploy_project."+projectName+".id}", dependencies)
		}

		if err != nil {
			return err
		}
	}

	// Export the triggers
	err = c.ProjectTriggerConverter.ToHclByProjectIdAndName(project.Id, project.Name, dependencies)

	if err != nil {
		return err
	}

	// Export the runbooks process
	if project.DeploymentProcessId != nil && !c.ExcludeAllRunbooks {
		var err error
		if lookup {
			err = c.RunbookConverter.ToHclLookupByIdAndName(project.Id, project.Name, dependencies)
		} else {
			err = c.RunbookConverter.ToHclByIdAndName(project.Id, project.Name, dependencies)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (c ProjectConverter) exportDependencyLookups(project octopus.Project, projectName string, dependencies *ResourceDetailsCollection) error {
	// Export the project group
	err := c.ProjectGroupConverter.ToHclLookupById(project.ProjectGroupId, dependencies)

	if err != nil {
		return err
	}

	// Export the library sets
	for _, v := range project.IncludedLibraryVariableSetIds {
		err := c.LibraryVariableSetConverter.ToHclLookupById(v, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the lifecycles
	err = c.LifecycleConverter.ToHclLookupById(project.LifecycleId, dependencies)

	if err != nil {
		return err
	}

	// Export the tenants
	err = c.TenantConverter.ToHclLookupByProjectId(project.Id, dependencies)

	if err != nil {
		return err
	}

	// Export the git credentials
	if project.PersistenceSettings.Credentials.Type == "Reference" {
		err = c.GitCredentialsConverter.ToHclLookupById(project.PersistenceSettings.Credentials.Id, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c ProjectConverter) exportDependencies(project octopus.Project, projectName string, dependencies *ResourceDetailsCollection) error {
	// Export the project group
	err := c.ProjectGroupConverter.ToHclById(project.ProjectGroupId, dependencies)

	if err != nil {
		return err
	}

	// Export the library sets
	for _, v := range project.IncludedLibraryVariableSetIds {
		err := c.LibraryVariableSetConverter.ToHclById(v, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the lifecycles
	err = c.LifecycleConverter.ToHclById(project.LifecycleId, dependencies)

	if err != nil {
		return err
	}

	// Export the tenants
	err = c.TenantConverter.ToHclByProjectId(project.Id, dependencies)

	if err != nil {
		return err
	}

	// Export the git credentials
	if project.PersistenceSettings.Credentials.Type == "Reference" {
		err = c.GitCredentialsConverter.ToHclById(project.PersistenceSettings.Credentials.Id, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}
