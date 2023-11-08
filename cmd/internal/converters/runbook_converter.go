package converters

import (
	"errors"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"k8s.io/utils/strings/slices"
	"regexp"
)

type RunbookConverter struct {
	Client                       client.OctopusClient
	RunbookProcessConverter      ConverterAndLookupByIdAndName
	EnvironmentConverter         ConverterAndLookupById
	ProjectConverter             ConverterAndLookupById
	ExcludedRunbooks             args.ExcludeRunbooks
	ExcludeRunbooksRegex         args.ExcludeRunbooks
	excludeRunbooksRegexCompiled []*regexp.Regexp
	IgnoreProjectChanges         bool
}

// ToHclByIdWithLookups exports a self-contained representation of the runbook where external resources like
// environments, lifecycles, feeds, accounts, projects etc are resolved with data lookups.
func (c *RunbookConverter) ToHclByIdWithLookups(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Runbook{}
	foundRunbook, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if !foundRunbook {
		return errors.New("failed to find runbook with id " + id)
	}

	parentResource := octopus.Project{}
	foundProject, err := c.Client.GetResourceById("Projects", resource.ProjectId, &parentResource)

	if err != nil {
		return err
	}

	if !foundProject {
		return errors.New("failed to find project with id " + resource.ProjectId)
	}

	zap.L().Info("Runbook: " + resource.Id)
	return c.toHcl(resource, parentResource.Name, false, true, dependencies)
}

func (c *RunbookConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Runbook{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	parentResource := octopus.Project{}
	_, err = c.Client.GetResourceById(c.GetResourceType(), id, &parentResource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, parentResource.Name, true, false, dependencies)
}

func (c *RunbookConverter) ToHclByIdAndName(projectId string, projectName string, dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Runbook]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Runbook: " + resource.Id)
		err = c.toHcl(resource, projectName, true, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *RunbookConverter) ToHclLookupByIdAndName(projectId string, projectName string, dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Runbook]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Runbook: " + resource.Id)
		err = c.toHcl(resource, projectName, false, true, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *RunbookConverter) toHcl(runbook octopus.Runbook, projectName string, recursive bool, lookups bool, dependencies *ResourceDetailsCollection) error {
	c.compileRegexes()

	if c.runbookIsExcluded(runbook) {
		return nil
	}

	thisResource := ResourceDetails{}

	resourceNameSuffix := sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(runbook.Name)
	runbookName := "runbook_" + resourceNameSuffix

	err := c.exportChildDependencies(recursive, lookups, runbook, resourceNameSuffix, dependencies)

	if err != nil {
		return err
	}

	thisResource.FileName = "space_population/" + runbookName + ".tf"
	thisResource.Id = runbook.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_runbook." + runbookName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformRunbook{
			Type:                     "octopusdeploy_runbook",
			Name:                     runbookName,
			ResourceName:             "${var." + runbookName + "_name}",
			ProjectId:                dependencies.GetResource("Projects", runbook.ProjectId),
			EnvironmentScope:         runbook.EnvironmentScope,
			Environments:             dependencies.GetResources("Environments", runbook.Environments...),
			ForcePackageDownload:     runbook.ForcePackageDownload,
			DefaultGuidedFailureMode: runbook.DefaultGuidedFailureMode,
			Description:              runbook.Description,
			MultiTenancyMode:         runbook.MultiTenancyMode,
			RetentionPolicy:          c.convertRetentionPolicy(runbook),
			ConnectivityPolicy:       c.convertConnectivityPolicy(runbook),
		}

		file := hclwrite.NewEmptyFile()

		c.writeProjectNameVariable(file, runbookName, runbook.Name)

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
			Type: hclsyntax.TokenComment,
			Bytes: []byte("# Import existing resources with the following commands:\n" +
				"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + runbook.Name + "\") | .Id')\n" +
				"# terraform import octopusdeploy_runbook." + runbookName + " ${RESOURCE_ID}\n"),
			SpacesBefore: 0,
		}})

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)

	return nil
}

func (c *RunbookConverter) GetResourceType() string {
	return "Runbooks"
}

func (c *RunbookConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/runbooks"
}

func (c *RunbookConverter) writeProjectNameVariable(file *hclwrite.File, projectName string, projectResourceName string) {
	runbookNameVariableResource := terraform.TerraformVariable{
		Name:        projectName + "_name",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The name of the runbook exported from " + projectResourceName,
		Default:     &projectResourceName,
	}

	block := gohcl.EncodeAsBlock(runbookNameVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)
}

func (c *RunbookConverter) convertConnectivityPolicy(runbook octopus.Runbook) *terraform.TerraformConnectivityPolicy {
	return &terraform.TerraformConnectivityPolicy{
		AllowDeploymentsToNoTargets: runbook.ConnectivityPolicy.AllowDeploymentsToNoTargets,
		ExcludeUnhealthyTargets:     runbook.ConnectivityPolicy.ExcludeUnhealthyTargets,
		SkipMachineBehavior:         runbook.ConnectivityPolicy.SkipMachineBehavior,
	}
}

func (c *RunbookConverter) convertRetentionPolicy(runbook octopus.Runbook) *terraform.RetentionPolicy {
	return &terraform.RetentionPolicy{
		QuantityToKeep:    runbook.RunRetentionPolicy.QuantityToKeep,
		ShouldKeepForever: runbook.RunRetentionPolicy.ShouldKeepForever,
	}
}

func (c *RunbookConverter) exportChildDependencies(recursive bool, lookup bool, runbook octopus.Runbook, runbookName string, dependencies *ResourceDetailsCollection) error {
	// It is not valid to have lookup be false and recursive be true, as the only supported export of a runbook is
	// with lookup being true.
	if lookup && recursive {
		return errors.New("exporting a runbook with dependencies is not supported")
	}

	// When lookup is true and recursive is false this runbook has been exported as a standalone resource
	// that references its parent project by a lookup.
	// If lookup is true and recursive is true, this runbook was exported with a project, and the project has already
	// been resolved.
	if lookup && !recursive && c.ProjectConverter != nil {
		err := c.ProjectConverter.ToHclLookupById(runbook.ProjectId, dependencies)

		if err != nil {
			return err
		}
	}

	// Export the deployment process
	if runbook.RunbookProcessId != nil {
		var err error
		if lookup {
			err = c.RunbookProcessConverter.ToHclLookupByIdAndName(*runbook.RunbookProcessId, runbookName, dependencies)
		} else {
			err = c.RunbookProcessConverter.ToHclByIdAndName(*runbook.RunbookProcessId, runbookName, dependencies)
		}

		if err != nil {
			return err
		}
	}

	for _, e := range runbook.Environments {
		var err error
		if recursive {
			err = c.EnvironmentConverter.ToHclById(e, dependencies)
		} else if lookup {
			err = c.EnvironmentConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *RunbookConverter) compileRegexes() {
	if c.ExcludeRunbooksRegex != nil {
		c.excludeRunbooksRegexCompiled = lo.FilterMap(c.ExcludeRunbooksRegex, func(x string, index int) (*regexp.Regexp, bool) {
			re, err := regexp.Compile(x)
			if err != nil {
				return nil, false
			}
			return re, true
		})
	}
}

func (c *RunbookConverter) runbookIsExcluded(runbook octopus.Runbook) bool {
	if c.ExcludedRunbooks != nil && slices.Index(c.ExcludedRunbooks, runbook.Name) != -1 {
		return true
	}

	if c.excludeRunbooksRegexCompiled != nil {
		return lo.SomeBy(c.excludeRunbooksRegexCompiled, func(x *regexp.Regexp) bool {
			return x.MatchString(runbook.Name)
		})
	}

	return false
}
