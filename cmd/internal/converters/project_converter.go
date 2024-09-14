package converters

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"strings"
)

const octopusdeployProjectsDataType = "octopusdeploy_projects"
const octopusdeployProjectResourceType = "octopusdeploy_project"

type ProjectConverter struct {
	Client                      client.OctopusClient
	LifecycleConverter          ConverterAndLookupWithStatelessById
	GitCredentialsConverter     ConverterAndLookupWithStatelessById
	LibraryVariableSetConverter ConverterAndLookupWithStatelessById
	ProjectGroupConverter       ConverterAndLookupWithStatelessById
	DeploymentProcessConverter  ConverterAndLookupByIdAndNameOrBranch
	TenantConverter             ConverterAndLookupByProjectId
	ProjectTriggerConverter     ConverterByProjectIdWithName
	VariableSetConverter        ConverterAndLookupByProjectIdAndName
	ChannelConverter            ConverterAndLookupByProjectIdWithTerraDependencies
	RunbookConverter            ConverterAndLookupByIdAndName
	EnvironmentConverter        ConverterAndLookupWithStatelessById
	IgnoreCacManagedValues      bool
	ExcludeCaCProjectSettings   bool
	ExcludeAllRunbooks          bool
	IgnoreProjectChanges        bool
	IgnoreProjectGroupChanges   bool
	IgnoreProjectNameChanges    bool
	ExcludeProjects             args.StringSliceArgs
	ExcludeProjectsExcept       args.StringSliceArgs
	ExcludeProjectsRegex        args.StringSliceArgs
	ExcludeAllProjects          bool
	DummySecretVariableValues   bool
	DummySecretGenerator        dummy.DummySecretGenerator
	Excluder                    ExcludeByName
	// This is set to true when this converter is only to be used to call ToHclLookupById
	LookupOnlyMode            bool
	ErrGroup                  *errgroup.Group
	ExcludeTerraformVariables bool
	IncludeIds                bool
	LimitResourceCount        int
	IncludeSpaceInPopulation  bool
	GenerateImportScripts     bool
	LookupProjectLinkTenants  bool
	TenantProjectConverter    TenantProjectConverter
	TenantVariableConverter   ToHclByTenantIdAndProject
	ExcludeTenantTagSets      args.StringSliceArgs
	ExcludeTenantTags         args.StringSliceArgs
	ExcludeTenants            args.StringSliceArgs
	ExcludeTenantsRegex       args.StringSliceArgs
	ExcludeTenantsWithTags    args.StringSliceArgs
	ExcludeTenantsExcept      args.StringSliceArgs
	ExcludeAllTenants         bool
	IgnoreCacErrors           bool
}

func (c *ProjectConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c *ProjectConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c *ProjectConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.LookupOnlyMode {
		return errors.New("this function can not be called whe LookupOnlyMode is true")
	}

	if c.ExcludeAllProjects {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.Project]{
		Client: c.Client,
	}

	done := make(chan struct{})
	defer close(done)

	channel := batchClient.GetAllResourcesBatch(done, c.GetResourceType())

	for resourceWrapper := range channel {
		if resourceWrapper.Err != nil {
			return resourceWrapper.Err
		}

		resource := resourceWrapper.Res
		if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllProjects, c.ExcludeProjects, c.ExcludeProjectsRegex, c.ExcludeProjectsExcept) {
			continue
		}

		c.ErrGroup.Go(func() error {
			zap.L().Info("Project: " + resource.Id)
			return c.toHcl(resource, false, false, stateless, dependencies)
		})

	}

	return nil
}

func (c *ProjectConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	project := octopus.Project{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	thisResource := data.ResourceDetails{}

	thisResource.FileName = "space_population/parent_project.tf"
	thisResource.Id = project.Id
	thisResource.Name = project.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployProjectsDataType + ".parent_project.projects[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData("parent_project", "${var.parent_project_name}")
		projectNameVariable := terraform.TerraformVariable{
			Name:        "parent_project_name",
			Type:        "string",
			Nullable:    false,
			Sensitive:   false,
			Description: "The name of the project to attach the runbook to",
			Default:     &project.Name,
		}

		file := hclwrite.NewEmptyFile()

		variableBlock := gohcl.EncodeAsBlock(projectNameVariable, "variable")
		hcl.WriteUnquotedAttribute(variableBlock, "type", "string")
		file.Body().AppendBlock(variableBlock)

		dataBlock := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(dataBlock, "Failed to resolve an project called \""+project.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.projects) != 0")
		file.Body().AppendBlock(dataBlock)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

// ToHclByIdWithLookups exports a self-contained representation of the project where external resources like
// environments, lifecycles, feeds, accounts etc are resolved with data lookups.
func (c *ProjectConverter) ToHclByIdWithLookups(id string, dependencies *data.ResourceDetailsCollection) error {
	if c.LookupOnlyMode {
		return errors.New("this function can not be called whe LookupOnlyMode is true")
	}

	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Project{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	zap.L().Info("Project: " + resource.Id)
	return c.toHcl(resource, false, true, false, dependencies)
}

func (c *ProjectConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	project := octopus.Project{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	return c.toHcl(project, true, false, true, dependencies)
}

func (c *ProjectConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	project := octopus.Project{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	return c.toHcl(project, true, false, false, dependencies)
}

func (c *ProjectConverter) buildData(resourceName string, name string) terraform.TerraformProjectData {
	return terraform.TerraformProjectData{
		Type:        octopusdeployProjectsDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c *ProjectConverter) writeData(file *hclwrite.File, name string, resourceName string) {
	terraformResource := c.buildData(resourceName, name)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

// toBashImport creates a bash script to import the resource
func (c *ProjectConverter) toBashImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
	dependencies.AddResource(data.ResourceDetails{
		FileName: "space_population/import_" + resourceName + ".sh",
		ToHcl: func() (string, error) {
			return fmt.Sprintf(`#!/bin/bash

# This script is used to import an exiting resource into the Terraform state.
# It is useful when importing a Terraform module into an Octopus space that
# already has existing resources.

# Make the script executable with the command:
# chmod +x ./import_%s.sh

# Alternativly, run the script with bash directly:
# /bin/bash ./import_%s.sh <options>

# Run "terraform init" to download any required providers and to configure the
# backend configuration

# Then run the import script. Replace the API key, instance URL, and Space ID 
# in the example below with the values of the space that the Terraform module 
# will be imported into.

# ./import_%s.sh API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234

if [[ $# -ne 3 ]]
then
	echo "Usage: ./import_%s.sh <API Key> <Octopus URL> <Space ID>"
    echo "Example: ./import_%s.sh API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234"
	exit 1
fi

if ! command -v jq &> /dev/null
then
    echo "jq is required"
    exit 1
fi

if ! command -v curl &> /dev/null
then
    echo "curl is required"
    exit 1
fi

RESOURCE_NAME="%s"
RESOURCE_ID=$(curl --silent -G --data-urlencode "partialName=${RESOURCE_NAME}" --data-urlencode "take=10000" --header "X-Octopus-ApiKey: $1" "$2/api/$3/Projects" | jq -r ".Items[] | select(.Name == \"${RESOURCE_NAME}\") | .Id")

if [[ -z RESOURCE_ID ]]
then
	echo "No project found with the name ${RESOURCE_NAME}"
	exit 1
fi

echo "Importing project ${RESOURCE_ID}"

terraform import "-var=octopus_server=$2" "-var=octopus_apikey=$1" "-var=octopus_space_id=$3" %s.%s ${RESOURCE_ID}`, resourceName, resourceName, resourceName, resourceName, resourceName, projectName, octopusdeployProjectResourceType, resourceName), nil
		},
	})
}

// toPowershellImport creates a powershell script to import the resource
func (c *ProjectConverter) toPowershellImport(resourceName string, projectName string, dependencies *data.ResourceDetailsCollection) {
	dependencies.AddResource(data.ResourceDetails{
		FileName: "space_population/import_" + resourceName + ".ps1",
		ToHcl: func() (string, error) {
			return fmt.Sprintf(`# This script is used to import an exiting resource into the Terraform state.
# It is useful when importing a Terraform module into an Octopus space that
# already has existing resources.

# Run "terraform init" to download any required providers and to configure the
# backend configuration

# Then run the import script. Replace the API key, instance URL, and Space ID 
# in the example below with the values of the space that the Terraform module 
# will be imported into.

# ./import_%s.ps1 API-xxxxxxxxxxxx https://yourinstance.octopus.app Spaces-1234

param (
    [Parameter(Mandatory=$true)]
    [string]$ApiKey,

    [Parameter(Mandatory=$true)]
    [string]$Url,

    [Parameter(Mandatory=$true)]
    [string]$SpaceId
)

$ResourceName="%s"

$headers = @{
    "X-Octopus-ApiKey" = $ApiKey
}

$ResourceId = Invoke-RestMethod -Uri "$Url/api/$SpaceId/Projects?take=10000&partialName=$([System.Web.HttpUtility]::UrlEncode($ResourceName))" -Method Get -Headers $headers |
	Select-Object -ExpandProperty Items | 
	Where-Object {$_.Name -eq $ResourceName} | 
	Select-Object -ExpandProperty Id

if ([System.String]::IsNullOrEmpty($ResourceId)) {
	echo "No project found with the name $ResourceName"
	exit 1
}

echo "Importing project $ResourceId"

terraform import "-var=octopus_server=$Url" "-var=octopus_apikey=$ApiKey" "-var=octopus_space_id=$SpaceId" %s.%s $ResourceId`, resourceName, projectName, octopusdeployProjectResourceType, resourceName), nil
		},
	})
}

func (c *ProjectConverter) toHcl(project octopus.Project, recursive bool, lookups bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded projects
	if c.Excluder.IsResourceExcludedWithRegex(project.Name, c.ExcludeAllProjects, c.ExcludeProjects, c.ExcludeProjectsRegex, c.ExcludeProjectsExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + project.Id)
		return nil
	}

	thisResource := data.ResourceDetails{}

	projectName := "project_" + sanitizer.SanitizeName(project.Name)

	if recursive {
		if err := c.exportDependencies(project, stateless, dependencies); err != nil {
			return err
		}
	} else if lookups {
		if err := c.exportDependencyLookups(project, dependencies); err != nil {
			return err
		}

		if err := c.linkTenantsAndCreateVars(project, stateless, dependencies); err != nil {
			return err
		}

	}

	if err := c.exportChildDependencies(recursive, lookups, stateless, project, projectName, dependencies); err != nil {
		return err
	}

	if c.GenerateImportScripts {
		c.toBashImport(projectName, project.Name, dependencies)
		c.toPowershellImport(projectName, project.Name, dependencies)
	}

	// The templates are dependencies that we export as part of the project
	projectTemplates, variables, projectTemplateMap := c.convertTemplates(project.Templates, projectName, stateless)
	dependencies.AddResource(projectTemplateMap...)

	thisResource.Parameters = c.getStepTemplateParameters(projectName, project, dependencies)
	thisResource.FileName = "space_population/project_" + projectName + ".tf"
	thisResource.Id = project.Id
	thisResource.Name = project.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${" + octopusdeployProjectResourceType + "." + projectName + ".id}"

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + projectName + ".projects) != 0 " +
			"? data." + octopusdeployProjectsDataType + "." + projectName + ".projects[0].id " +
			": " + octopusdeployProjectResourceType + "." + projectName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployProjectResourceType + "." + projectName + "}"
		thisResource.Count = "${length(data." + octopusdeployProjectsDataType + "." + projectName + ".projects) != 0 ? 0 : 1}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProjectResourceType + "." + projectName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		resourceName := c.writeProjectNameVariable(file, projectName, project.Name)
		description := c.writeProjectDescriptionVariable(file, projectName, project.Name, strutil.EmptyIfNil(project.Description))
		tenanted := c.writeProjectTenantedVariable(file, projectName, strutil.EmptyIfNil(project.TenantedDeploymentMode))

		// If we are excluding version controlled settings, the version controlled field will be false
		versionControlled := project.IsVersionControlled
		if c.ExcludeCaCProjectSettings {
			versionControlled = false
		}

		terraformResource := terraform.TerraformProject{
			Type:                                   octopusdeployProjectResourceType,
			Name:                                   projectName,
			ResourceName:                           resourceName,
			Id:                                     strutil.InputPointerIfEnabled(c.IncludeIds, &project.Id),
			SpaceId:                                strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", project.SpaceId)),
			AutoCreateRelease:                      false, // TODO: Would be project.AutoCreateRelease, but there is no way to reference the package
			DefaultGuidedFailureMode:               project.DefaultGuidedFailureMode,
			DefaultToSkipIfAlreadyInstalled:        project.DefaultToSkipIfAlreadyInstalled,
			DiscreteChannelRelease:                 project.DiscreteChannelRelease,
			IsDisabled:                             project.IsDisabled,
			IsVersionControlled:                    versionControlled,
			LifecycleId:                            dependencies.GetResource("Lifecycles", project.LifecycleId),
			ProjectGroupId:                         dependencies.GetResource("ProjectGroups", project.ProjectGroupId),
			IncludedLibraryVariableSets:            c.convertLibraryVariableSets(project.IncludedLibraryVariableSetIds, dependencies),
			TenantedDeploymentParticipation:        &tenanted,
			Template:                               projectTemplates,
			ConnectivityPolicy:                     c.convertConnectivityPolicy(project),
			GitLibraryPersistenceSettings:          c.convertLibraryGitPersistence(project, projectName, dependencies),
			GitAnonymousPersistenceSettings:        c.convertAnonymousGitPersistence(project, projectName),
			GitUsernamePasswordPersistenceSettings: c.convertUsernamePasswordGitPersistence(project, projectName),
			VersioningStrategy:                     c.convertVersioningStrategy(project),
		}

		// There is no point ignoring changes for stateless exports
		if !c.IgnoreProjectChanges && !stateless {
			ignoreList := []string{}

			if project.HasCacConfigured() {
				ignoreList = append(ignoreList, "connectivity_policy")
			}

			if c.IgnoreProjectGroupChanges {
				ignoreList = append(ignoreList, "project_group_id")
			}

			if c.IgnoreProjectNameChanges {
				ignoreList = append(ignoreList, "name")
			}

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues {
				ignoreList = append(ignoreList, "git_username_password_persistence_settings[0].password")
			}

			terraformResource.Lifecycle = &terraform.TerraformLifecycleMetaArgument{
				IgnoreChanges: &ignoreList,
			}
		}

		if stateless {
			c.writeData(file, "${var."+projectName+"_name}", projectName)
			terraformResource.Count = strutil.StrPointer(thisResource.Count)
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")
		hcl.WriteUnquotedAttribute(block, "description", description)

		// write any variables used to define the value of tenant template secrets
		for _, variable := range variables {
			block := gohcl.EncodeAsBlock(variable, "variable")
			hcl.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)
		}

		if !c.ExcludeCaCProjectSettings {
			if terraformResource.GitUsernamePasswordPersistenceSettings != nil {
				secretVariableResource := terraform.TerraformVariable{
					Name:        projectName + "_git_password",
					Type:        "string",
					Nullable:    false,
					Sensitive:   true,
					Description: "The git password for the project \"" + project.Name + "\"",
				}

				if c.DummySecretVariableValues {
					secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
					dependencies.AddDummy(data.DummyVariableReference{
						VariableName: projectName + "_git_password",
						ResourceName: project.Name,
						ResourceType: c.GetResourceType(),
					})
				}

				block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
				hcl.WriteUnquotedAttribute(block, "type", "string")
				file.Body().AppendBlock(block)
			}

			if terraformResource.HasCacConfigured() {
				c.writeGitPathVar(projectName, project, file)
				c.writeGitUrlVar(projectName, project, file)
				c.writeProtectedBranchesVar(projectName, project, file)
			}
		}

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		file.Body().AppendBlock(block)
		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)

	return nil
}

func (c *ProjectConverter) writeGitUrlVar(projectName string, project octopus.Project, file *hclwrite.File) {
	variableResource := terraform.TerraformVariable{
		Name:        projectName + "_git_url",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The git url for \"" + project.Name + "\"",
		Default:     &project.PersistenceSettings.Url,
	}

	block := gohcl.EncodeAsBlock(variableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)
}

func (c *ProjectConverter) getStepTemplateParameters(projectName string, project octopus.Project, dependencies *data.ResourceDetailsCollection) []data.ResourceParameter {
	parameters := []data.ResourceParameter{}

	if project.PersistenceSettings.Credentials.Type == "UsernamePassword" && !c.ExcludeCaCProjectSettings {
		parameters = append(parameters, data.ResourceParameter{
			VariableName:  projectName + "_git_password",
			Label:         "Project " + project.Name + " Git password",
			Description:   "The Git password associated with the project \"" + project.Name + "\"",
			ResourceName:  sanitizer.SanitizeParameterName(dependencies, project.Name, "GitPassword"),
			ParameterType: "GitPassword",
			Sensitive:     true,
		})
	}

	for _, v := range project.Templates {
		if setting, ok := v.DisplaySettings["Octopus.ControlType"]; ok && setting == "Sensitive" {
			variableName := sanitizer.SanitizeName(projectName + "_template_" + strutil.EmptyIfNil(v.Name))
			parameters = append(parameters, data.ResourceParameter{
				VariableName:  variableName,
				Label:         "Project " + project.Name + " tenant variable " + strutil.EmptyIfNil(v.Name),
				Description:   "Sensitive value for tenant variable template " + strutil.EmptyIfNil(v.Name) + " for project " + project.Name,
				ResourceName:  sanitizer.SanitizeParameterName(dependencies, project.Name, "GitPassword"),
				Sensitive:     true,
				ParameterType: "GitPassword",
			})
		}
	}

	return parameters
}

// Octopus has two places where protected branches are defined: the explicit list of branches and optionally the default branch.
// The TF provider only has the explicit list. So getProtectedBranches builds a single list taking these two sources
// into account.
func (c *ProjectConverter) getProtectedBranches(project octopus.Project) []string {
	branches := lo.Map(project.PersistenceSettings.ProtectedBranchNamePatterns, func(x string, index int) string {
		return strings.ReplaceAll(x, "\"", "\\\"")
	})

	if project.PersistenceSettings.ProtectedDefaultBranch {
		branches = append(branches, project.PersistenceSettings.DefaultBranch)
	}

	return branches
}

func (c *ProjectConverter) writeProtectedBranchesVar(projectName string, project octopus.Project, file *hclwrite.File) {

	sanitizedList := c.getProtectedBranches(project)

	if project.PersistenceSettings.ProtectedDefaultBranch {
		sanitizedList = []string{project.PersistenceSettings.DefaultBranch}
	} else {
		sanitizedList = lo.Map(project.PersistenceSettings.ProtectedBranchNamePatterns, func(x string, index int) string {
			return strings.ReplaceAll(x, "\"", "\\\"")
		})
	}

	list := "[]"
	if len(sanitizedList) != 0 {
		list = "[\"" + strings.Join(sanitizedList, "\", ") + "\"]"
	}

	variableResource := terraform.TerraformVariable{
		Name:        projectName + "_git_protected_branches",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The protected branches for \"" + project.Name + "\"",
		Default:     &list,
	}

	block := gohcl.EncodeAsBlock(variableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)
}

func (c *ProjectConverter) writeGitPathVar(projectName string, project octopus.Project, file *hclwrite.File) {
	variableResource := terraform.TerraformVariable{
		Name:        projectName + "_git_base_path",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The git base path for \"" + project.Name + "\"",
		Default:     &project.PersistenceSettings.BasePath,
	}

	block := gohcl.EncodeAsBlock(variableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)
}

func (c *ProjectConverter) GetResourceType() string {
	return "Projects"
}

func (c *ProjectConverter) writeProjectNameVariable(file *hclwrite.File, projectName string, projectResourceName string) string {
	if c.ExcludeTerraformVariables {
		return projectResourceName
	}

	sanitizedProjectName := sanitizer.SanitizeName(projectName)

	secretVariableResource := terraform.TerraformVariable{
		Name:        sanitizedProjectName + "_name",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The name of the project exported from " + projectResourceName,
		Default:     &projectResourceName,
	}

	block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	return "${var." + projectName + "_name}"
}

func (c *ProjectConverter) writeProjectTenantedVariable(file *hclwrite.File, projectName string, tenantedSetting string) string {
	if c.ExcludeTerraformVariables {
		return tenantedSetting
	}

	sanitizedProjectName := sanitizer.SanitizeName(projectName)

	secretVariableResource := terraform.TerraformVariable{
		Name:        sanitizedProjectName + "_tenanted",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The tenanted setting for the project " + tenantedSetting,
		Default:     &tenantedSetting,
	}

	block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	return "${var." + projectName + "_tenanted}"
}

func (c *ProjectConverter) writeProjectDescriptionVariable(file *hclwrite.File, projectResourceName string, projectName string, projectResourceDescription string) string {
	if c.ExcludeTerraformVariables {
		escapedStr, err := json.Marshal(projectResourceDescription)
		if err != nil {
			return ""
		}

		return string(escapedStr)
	}

	sanitizedProjectName := sanitizer.SanitizeName(projectResourceName)

	descriptionPrefixVariable := terraform.TerraformVariable{
		Name:        sanitizedProjectName + "_description_prefix",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "An optional prefix to add to the project description for the project " + projectName,
		Default:     strutil.StrPointer(""),
	}

	prefixBlock := gohcl.EncodeAsBlock(descriptionPrefixVariable, "variable")
	hcl.WriteUnquotedAttribute(prefixBlock, "type", "string")
	file.Body().AppendBlock(prefixBlock)

	descriptionSuffixVariable := terraform.TerraformVariable{
		Name:        sanitizedProjectName + "_description_suffix",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "An optional suffix to add to the project description for the project " + projectName,
		Default:     strutil.StrPointer(""),
	}

	suffixBlock := gohcl.EncodeAsBlock(descriptionSuffixVariable, "variable")
	hcl.WriteUnquotedAttribute(suffixBlock, "type", "string")
	file.Body().AppendBlock(suffixBlock)

	/*
		The default value wraps the existing project description with a prefix and suffix to allow the final
		description to be easily modified.
	*/
	descriptionVariable := terraform.TerraformVariable{
		Name:        sanitizedProjectName + "_description",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The description of the project exported from " + projectName,
		Default:     strutil.StrPointer(projectResourceDescription),
	}

	block := gohcl.EncodeAsBlock(descriptionVariable, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	return "\"${var." + sanitizedProjectName + "_description_prefix}${var." + sanitizedProjectName + "_description}${var." + sanitizedProjectName + "_description_suffix}\""
}

func (c *ProjectConverter) convertTemplates(actionPackages []octopus.Template, projectName string, stateless bool) ([]terraform.TerraformTemplate, []terraform.TerraformVariable, []data.ResourceDetails) {
	templateMap := make([]data.ResourceDetails, 0)
	collection := make([]terraform.TerraformTemplate, 0)
	variables := []terraform.TerraformVariable{}
	for i, v := range actionPackages {
		if setting, ok := v.DisplaySettings["Octopus.ControlType"]; ok && setting == "Sensitive" {
			variableName := sanitizer.SanitizeName(projectName + "_template_" + strutil.EmptyIfNil(v.Name))

			secretVariableResource := terraform.TerraformVariable{
				Name:        variableName,
				Type:        "string",
				Nullable:    false,
				Sensitive:   true,
				Description: "Sensitive value for tenant variable template " + strutil.EmptyIfNil(v.Name),
				Default:     strutil.StrPointer("replace me with a password"),
			}

			variables = append(variables, secretVariableResource)

			collection = append(collection, terraform.TerraformTemplate{
				Name:     v.Name,
				Label:    v.Label,
				HelpText: v.HelpText,
				// Is this a bug? This may need to have a field for sensitive values, but the provider does
				// not expose that today.
				DefaultValue:    strutil.StrPointer("${var." + variableName + "}"),
				DisplaySettings: v.DisplaySettings,
			})
		} else {
			collection = append(collection, terraform.TerraformTemplate{
				Name:            v.Name,
				Label:           v.Label,
				HelpText:        v.HelpText,
				DefaultValue:    strutil.EscapeDollarCurlyPointer(v.GetDefaultValueString()),
				DisplaySettings: v.DisplaySettings,
			})
		}

		templateMap = append(templateMap, data.ResourceDetails{
			Id:           v.Id,
			ResourceType: "ProjectTemplates",
			Lookup:       "${" + octopusdeployProjectResourceType + "." + projectName + ".template[" + fmt.Sprint(i) + "].id}",
			FileName:     "",
			ToHcl:        nil,
		})
	}
	return collection, variables, templateMap
}

func (c *ProjectConverter) getLookup(stateless bool, projectName string, index int) string {
	if stateless {
		// There is no tag lookup, so if the project exists, the template is not created, and the lookup is an
		// empty string.
		return "${length(data." + octopusdeployProjectsDataType + "." + projectName + ".projects) != 0 " +
			"? '' " +
			": ${" + octopusdeployProjectResourceType + "." + projectName + ".template[" + fmt.Sprint(index) + "][0].id}"
	}
	return "${" + octopusdeployProjectResourceType + "." + projectName + ".template[" + fmt.Sprint(index) + "].id}"
}

func (c *ProjectConverter) convertConnectivityPolicy(project octopus.Project) *terraform.TerraformConnectivityPolicy {
	if c.IgnoreCacManagedValues && project.HasCacConfigured() {
		return nil
	}

	return &terraform.TerraformConnectivityPolicy{
		AllowDeploymentsToNoTargets: project.ProjectConnectivityPolicy.AllowDeploymentsToNoTargets,
		ExcludeUnhealthyTargets:     project.ProjectConnectivityPolicy.ExcludeUnhealthyTargets,
		SkipMachineBehavior:         project.ProjectConnectivityPolicy.SkipMachineBehavior,
	}
}

func (c *ProjectConverter) convertLibraryVariableSets(setIds []string, dependencies *data.ResourceDetailsCollection) []string {
	collection := make([]string, 0)
	for _, v := range setIds {
		libraryVariableSet := dependencies.GetResource("LibraryVariableSets", v)
		if libraryVariableSet != "" {
			collection = append(collection, dependencies.GetResource("LibraryVariableSets", v))
		}
	}
	return collection
}

func (c *ProjectConverter) convertLibraryGitPersistence(project octopus.Project, projectName string, dependencies *data.ResourceDetailsCollection) *terraform.TerraformGitLibraryPersistenceSettings {
	if project.PersistenceSettings.Credentials.Type != "Reference" || c.ExcludeCaCProjectSettings {
		return nil
	}

	return &terraform.TerraformGitLibraryPersistenceSettings{
		GitCredentialId:   dependencies.GetResource("Git-Credentials", project.PersistenceSettings.Credentials.Id),
		Url:               "${var." + projectName + "_git_url}",
		BasePath:          "${var." + projectName + "_git_base_path}",
		DefaultBranch:     project.PersistenceSettings.DefaultBranch,
		ProtectedBranches: "${jsondecode(var." + projectName + "_git_protected_branches)}",
	}
}

func (c *ProjectConverter) convertAnonymousGitPersistence(project octopus.Project, projectName string) *terraform.TerraformGitAnonymousPersistenceSettings {
	if project.PersistenceSettings.Credentials.Type != "Anonymous" || c.ExcludeCaCProjectSettings {
		return nil
	}

	return &terraform.TerraformGitAnonymousPersistenceSettings{
		Url:               "${var." + projectName + "_git_url}",
		BasePath:          "${var." + projectName + "_git_base_path}",
		DefaultBranch:     project.PersistenceSettings.DefaultBranch,
		ProtectedBranches: "${jsondecode(var." + projectName + "_git_protected_branches)}",
	}
}

func (c *ProjectConverter) convertUsernamePasswordGitPersistence(project octopus.Project, projectName string) *terraform.TerraformGitUsernamePasswordPersistenceSettings {
	if project.PersistenceSettings.Credentials.Type != "UsernamePassword" || c.ExcludeCaCProjectSettings {
		return nil
	}

	return &terraform.TerraformGitUsernamePasswordPersistenceSettings{
		Url:               "${var." + projectName + "_git_url}",
		Username:          project.PersistenceSettings.Credentials.Username,
		Password:          "${var." + projectName + "_git_password}",
		BasePath:          "${var." + projectName + "_git_base_path}",
		DefaultBranch:     project.PersistenceSettings.DefaultBranch,
		ProtectedBranches: "${jsondecode(var." + projectName + "_git_protected_branches)}",
	}
}

func (c *ProjectConverter) convertVersioningStrategy(project octopus.Project) *terraform.TerraformVersioningStrategy {
	if c.IgnoreCacManagedValues && project.HasCacConfigured() {
		return nil
	}

	// Don't define a versioning strategy if it is not set
	if project.VersioningStrategy.Template == "" {
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
func (c *ProjectConverter) exportChildDependencies(recursive bool, lookup bool, stateless bool, project octopus.Project, projectName string, dependencies *data.ResourceDetailsCollection) error {
	var err error
	if lookup {
		err = c.ChannelConverter.ToHclLookupByProjectIdWithTerraDependencies(project.Id, map[string]string{
			"DeploymentProcesses": strutil.EmptyIfNil(project.DeploymentProcessId),
		}, dependencies)
	} else {
		if stateless {
			err = c.ChannelConverter.ToHclStatelessByProjectIdWithTerraDependencies(project.Id, map[string]string{
				"DeploymentProcesses": strutil.EmptyIfNil(project.DeploymentProcessId),
			}, dependencies)
		} else {
			err = c.ChannelConverter.ToHclByProjectIdWithTerraDependencies(project.Id, map[string]string{
				"DeploymentProcesses": strutil.EmptyIfNil(project.DeploymentProcessId),
			}, dependencies)
		}
	}

	if err != nil {
		return err
	}

	// Export the deployment process
	if project.DeploymentProcessId != nil && !(c.IgnoreCacManagedValues && project.HasCacConfigured()) {

		var err error
		if lookup {
			err = c.DeploymentProcessConverter.ToHclLookupByIdAndName(*project.DeploymentProcessId, projectName, dependencies)
		} else {
			if stateless {
				err = c.DeploymentProcessConverter.ToHclStatelessByIdAndName(*project.DeploymentProcessId, projectName, dependencies)
			} else {
				err = c.DeploymentProcessConverter.ToHclByIdAndName(*project.DeploymentProcessId, projectName, recursive, dependencies)
			}
		}

		if err != nil {
			return err
		}
	}

	// The deployment process for a CaC enabled project is found under the name of a Git branch
	if !c.IgnoreCacManagedValues && project.HasCacConfigured() {
		if lookup {
			err = c.DeploymentProcessConverter.ToHclLookupByIdAndBranch(project.Id, project.PersistenceSettings.DefaultBranch, dependencies)
		} else {
			if stateless {
				err = c.DeploymentProcessConverter.ToHclStatelessByIdAndBranch(project.Id, project.PersistenceSettings.DefaultBranch, dependencies)
			} else {
				err = c.DeploymentProcessConverter.ToHclByIdAndBranch(project.Id, project.PersistenceSettings.DefaultBranch, recursive, dependencies)
			}

		}

		if err != nil {
			return err
		}
	}

	var parentCount *string = nil
	var parentLookup = "${" + octopusdeployProjectResourceType + "." + projectName + ".id}"
	if stateless {
		parentCount = strutil.StrPointer("${length(data." + octopusdeployProjectsDataType + "." + projectName + ".projects) != 0 ? 0 : 1}")
		parentLookup = "${length(data." + octopusdeployProjectsDataType + "." + projectName + ".projects) == 0 ?" + octopusdeployProjectResourceType + "." + projectName + "[0].id : data.octopusdeploy_projects." + projectName + ".projects[0].id}"
	}

	// Export the variable set. Cac projects save secrets here, regular projects save all variables
	if project.VariableSetId != nil {
		var err error
		if lookup {
			err = c.VariableSetConverter.ToHclLookupByProjectIdAndName(
				project.Id,
				project.Name,
				"${"+octopusdeployProjectResourceType+"."+projectName+".id}", dependencies)
		} else {
			err = c.VariableSetConverter.ToHclByProjectIdAndName(
				project.Id,
				project.Name,
				parentLookup,
				parentCount,
				recursive,
				dependencies)
		}

		if err != nil {
			return err
		}
	}

	// The variables for a CaC enabled project are found under the name of a Git branch
	if !c.IgnoreCacManagedValues && project.HasCacConfigured() {
		if lookup {
			err = c.VariableSetConverter.ToHclLookupByProjectIdBranchAndName(
				project.Id,
				project.PersistenceSettings.DefaultBranch,
				project.Name,
				"${"+octopusdeployProjectResourceType+"."+projectName+".id}",
				dependencies)
		} else if stateless {
			err = c.VariableSetConverter.ToHclStatelessByProjectIdBranchAndName(
				project.Id,
				project.PersistenceSettings.DefaultBranch,
				project.Name,
				parentLookup,
				parentCount,
				recursive,
				dependencies)
		} else {
			err = c.VariableSetConverter.ToHclByProjectIdBranchAndName(
				project.Id,
				project.PersistenceSettings.DefaultBranch,
				project.Name,
				parentLookup,
				parentCount,
				recursive,
				dependencies)
		}

		if err != nil && !c.IgnoreCacErrors {
			return err
		}
	}

	// Export the triggers
	err = c.ProjectTriggerConverter.ToHclByProjectIdAndName(project.Id, project.Name, recursive, lookup, dependencies)

	if err != nil {
		return err
	}

	// Export the runbooks process
	if project.DeploymentProcessId != nil && !c.ExcludeAllRunbooks {
		var err error
		if lookup {
			err = c.RunbookConverter.ToHclLookupByIdAndName(project.Id, project.Name, dependencies)
		} else {
			if stateless {
				err = c.RunbookConverter.ToHclStatelessByIdAndName(project.Id, project.Name, dependencies)
			} else {
				err = c.RunbookConverter.ToHclByIdAndName(project.Id, project.Name, recursive, dependencies)
			}
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ProjectConverter) exportDependencyLookups(project octopus.Project, dependencies *data.ResourceDetailsCollection) error {
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

	// Export all environments (a tenant could link to any environment for runbooks, regardless of the lifecycles)
	environments := octopus.GeneralCollection[octopus.Environment]{}
	if err := c.Client.GetAllResources("Environments", &environments); err != nil {
		return fmt.Errorf("error in OctopusClient.GetAllResources loading type octopus.GeneralCollection[octopus.Environment]: %w", err)
	}

	for _, environment := range environments.Items {
		if err := c.EnvironmentConverter.ToHclLookupById(environment.Id, dependencies); err != nil {
			return err
		}
	}

	// Export the git credentials
	if project.PersistenceSettings.Credentials.Type == "Reference" && !c.ExcludeCaCProjectSettings {
		err = c.GitCredentialsConverter.ToHclLookupById(project.PersistenceSettings.Credentials.Id, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ProjectConverter) exportDependencies(project octopus.Project, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Export the project group
	if stateless {
		err := c.ProjectGroupConverter.ToHclStatelessById(project.ProjectGroupId, dependencies)

		if err != nil {
			return err
		}

		// Export the library sets
		for _, v := range project.IncludedLibraryVariableSetIds {
			err := c.LibraryVariableSetConverter.ToHclStatelessById(v, dependencies)

			if err != nil {
				return err
			}
		}

		// Export the lifecycles
		err = c.LifecycleConverter.ToHclStatelessById(project.LifecycleId, dependencies)

		if err != nil {
			return err
		}

		// Export the tenants
		err = c.TenantConverter.ToHclStatelessByProjectId(project.Id, dependencies)

		if err != nil {
			return err
		}

		// Export the git credentials
		if project.PersistenceSettings.Credentials.Type == "Reference" && !c.ExcludeCaCProjectSettings {
			err = c.GitCredentialsConverter.ToHclStatelessById(project.PersistenceSettings.Credentials.Id, dependencies)

			if err != nil {
				return err
			}
		}
	} else {
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
		if project.PersistenceSettings.Credentials.Type == "Reference" && !c.ExcludeCaCProjectSettings {
			err = c.GitCredentialsConverter.ToHclById(project.PersistenceSettings.Credentials.Id, dependencies)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *ProjectConverter) linkTenantsAndCreateVars(project octopus.Project, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if !c.LookupProjectLinkTenants {
		return nil
	}

	// Find the project tenants
	collection := octopus.GeneralCollection[octopus.Tenant]{}
	err := c.Client.GetAllResources("Tenants", &collection, []string{"projectId", project.Id})

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetAllResources loading type octopus.GeneralCollection[octopus.Tenant]: %w", err)
	}

	for _, tenant := range collection.Items {

		// Ignore excluded tenants
		if c.Excluder.IsResourceExcludedWithRegex(tenant.Name, c.ExcludeAllTenants, c.ExcludeTenants, c.ExcludeTenantsRegex, c.ExcludeTenantsExcept) {
			continue
		}

		// Ignore tenants with excluded tags
		if c.ExcludeTenantsWithTags != nil && tenant.TenantTags != nil && lo.SomeBy(tenant.TenantTags, func(item string) bool {
			return lo.IndexOf(c.ExcludeTenantsWithTags, item) != -1
		}) {
			continue
		}

		// Link the tenants to the project
		if environmentIds, ok := tenant.ProjectEnvironments[project.Id]; ok {
			c.TenantProjectConverter.LinkTenantToProject(tenant, project, environmentIds, dependencies)
		}

		// Create the project tenant variables
		if err := c.TenantVariableConverter.ToHclByTenantIdAndProject(tenant.Id, project, dependencies); err != nil {
			return err
		}
	}

	return nil
}
