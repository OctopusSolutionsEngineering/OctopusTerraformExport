package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/boolutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/maputil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/regexes"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sliceutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/steps"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"net/url"
)

// terraformProcessStepBlock maps a Terraform process step to a block in HCL.
// A nil OctopusStep indicates a child step
// A nil OctopusAction indicates a parent step with child steps in the UI.
// If both OctopusStep and OctopusAction are not nil, this is a step with a single action,
type terraformProcessStepBlock struct {
	Step          *terraform.TerraformProcessStep
	OctopusStep   *octopus.Step
	OctopusAction *octopus.Action
	Block         *hclwrite.Block
}

// DeploymentProcessConverterV2 converts deployment processes for v1 of the Octopus Terraform provider.
type DeploymentProcessConverterV2 struct {
	Client                          client.OctopusClient
	OctopusActionProcessor          *OctopusActionProcessor
	IgnoreProjectChanges            bool
	WorkerPoolProcessor             OctopusWorkerPoolProcessor
	ExcludeTenantTags               args.StringSliceArgs
	ExcludeTenantTagSets            args.StringSliceArgs
	Excluder                        ExcludeByName
	TagSetConverter                 ConvertToHclByResource[octopus.TagSet]
	LimitAttributeLength            int
	ExcludeTerraformVariables       bool
	ExcludeAllSteps                 bool
	ExcludeSteps                    args.StringSliceArgs
	ExcludeStepsRegex               args.StringSliceArgs
	ExcludeStepsExcept              args.StringSliceArgs
	IgnoreInvalidExcludeExcept      bool
	ExperimentalEnableStepTemplates bool
	DummySecretGenerator            dummy.DummySecretGenerator
	DummySecretVariableValues       bool
	IgnoreCacErrors                 bool
}

func (c *DeploymentProcessConverterV2) SetActionProcessor(actionProcessor *OctopusActionProcessor) {
	c.OctopusActionProcessor = actionProcessor
}

func (c *DeploymentProcessConverterV2) ToHclByIdAndBranch(parentId string, branch string, recursive bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndBranch(parentId, branch, recursive, false, dependencies)
}

func (c *DeploymentProcessConverterV2) ToHclStatelessByIdAndBranch(parentId string, branch string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndBranch(parentId, branch, true, true, dependencies)
}

func (c *DeploymentProcessConverterV2) toHclByIdAndBranch(parentId string, branch string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if parentId == "" || branch == "" {
		return nil
	}

	if dependencies.HasResource(parentId+"/"+branch, c.GetResourceType()) {
		return nil
	}

	// Get the deployment process associated with the git branch
	resource := octopus.DeploymentProcess{}
	found, err := c.Client.GetResource("Projects/"+parentId+"/"+url.QueryEscape(branch)+"/deploymentprocesses", &resource)

	if err != nil {
		if !c.IgnoreCacErrors {
			return err
		} else {
			found = false
		}
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", resource.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	return c.toHcl(resource, project, project.HasCacConfigured(), recursive, false, stateless, project.Name, dependencies)
}

func (c *DeploymentProcessConverterV2) ToHclLookupByIdAndBranch(parentId string, branch string, dependencies *data.ResourceDetailsCollection) error {
	if parentId == "" || branch == "" {
		return nil
	}

	if dependencies.HasResource(parentId+"/"+branch, c.GetResourceType()) {
		return nil
	}

	// Get the deployment process associated with the git branch
	resource := octopus.DeploymentProcess{}
	found, err := c.Client.GetResource("Projects/"+parentId+"/"+url.QueryEscape(branch)+"/deploymentprocesses", &resource)

	if err != nil {
		if !c.IgnoreCacErrors {
			return err
		} else {
			found = false
		}
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", resource.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	return c.toHcl(resource, project, project.HasCacConfigured(), false, true, false, project.Name, dependencies)
}

func (c *DeploymentProcessConverterV2) ToHclByIdAndName(id string, _ string, recursive bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndName(id, "", recursive, false, dependencies)
}

func (c *DeploymentProcessConverterV2) ToHclStatelessByIdAndName(id string, _ string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndName(id, "", true, true, dependencies)
}

func (c *DeploymentProcessConverterV2) toHclByIdAndName(id string, _ string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.DeploymentProcess{}
	found, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.DeploymentProcess: %w", err)
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", resource.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	zap.L().Info("Deployment Process: " + resource.Id)
	return c.toHcl(resource, project, project.HasCacConfigured(), recursive, false, stateless, project.Name, dependencies)
}

func (c *DeploymentProcessConverterV2) ToHclLookupByIdAndName(id string, _ string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.DeploymentProcess{}
	found, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.DeploymentProcess: %w", err)
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	project := octopus.Project{}
	_, err = c.Client.GetSpaceResourceById("Projects", resource.ProjectId, &project)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Project: %w", err)
	}

	return c.toHcl(resource, project, project.HasCacConfigured(), false, true, false, project.Name, dependencies)
}

func (c *DeploymentProcessConverterV2) toHcl(resource octopus.DeploymentProcess, project octopus.Project, cac bool, recursive bool, lookup bool, stateless bool, projectName string, dependencies *data.ResourceDetailsCollection) error {
	resourceName := "process_" + sanitizer.SanitizeName(projectName)

	thisResource := data.ResourceDetails{}

	err := c.exportDependencies(recursive, lookup, stateless, resource, dependencies)

	if err != nil {
		return err
	}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_process." + resourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		validSteps := FilterSteps(
			resource.Steps,
			c.IgnoreInvalidExcludeExcept,
			c.Excluder,
			c.ExcludeAllSteps,
			c.ExcludeSteps,
			c.ExcludeStepsRegex,
			c.ExcludeStepsExcept)

		terraformProcessResource := terraform.TerraformProcess{
			Type:      "octopusdeploy_process",
			Name:      resourceName,
			Id:        nil,
			SpaceId:   nil,
			ProjectId: strutil.StrPointer(dependencies.GetResource("Projects", resource.ProjectId)),
			RunbookId: nil,
		}

		terraformProcessSteps := []terraformProcessStepBlock{}
		terraformProcessStepsChildren := []terraformProcessStepBlock{}

		file := hclwrite.NewEmptyFile()

		for _, s := range validSteps {
			terraformProcessStep := terraform.TerraformProcessStep{
				Type:                 "octopusdeploy_process_step",
				Name:                 "process_step_" + sanitizer.SanitizeNamePointer(s.Name),
				Id:                   nil,
				ResourceName:         strutil.EmptyIfNil(s.Name),
				ResourceType:         "",
				ProcessId:            "${octopusdeploy_process." + terraformProcessResource.Name + ".id}",
				Channels:             nil,
				Condition:            s.Condition,
				Container:            nil,
				Environments:         nil,
				ExcludedEnvironments: nil,
				ExecutionProperties:  nil,
				GitDependencies:      nil,
				IsDisabled:           nil,
				IsRequired:           nil,
				Notes:                nil,
				Packages:             nil,
				PrimaryPackage:       nil,
				Slug:                 nil,
				SpaceId:              nil,
				TenantTags:           nil,
				WorkerPoolId:         nil,
				WorkerPoolVariable:   nil,
				StartTrigger:         s.StartTrigger,
				Properties:           nil,
				PackageRequirement:   s.PackageRequirement,
			}

			// We build the output differently for a step with a single action (represented as a typical step in the UI)
			// and a step with multiple actions (represented as a parent step with child steps in the UI).

			if len(s.Actions) == 1 {
				action := s.Actions[0]

				// The step type is the type of the first action.
				terraformProcessStep.ResourceType = strutil.EmptyIfNil(action.ActionType)

				c.assignPrimaryPackage(projectName, &terraformProcessStep, &action, file, dependencies)
				c.assignReferencePackage(projectName, &terraformProcessStep, &action, file, dependencies)
				if err := c.assignWorkerPool(&terraformProcessStep, &action, file, dependencies); err != nil {
					return "", err
				}
				terraformProcessStep.WorkerPoolVariable = strutil.NilIfEmptyPointer(action.WorkerPoolVariable)
				terraformProcessStep.Environments = sliceutil.NilIfEmpty(dependencies.GetResources("Environments", action.Environments...))
				terraformProcessStep.ExcludedEnvironments = sliceutil.NilIfEmpty(dependencies.GetResources("Environments", action.ExcludedEnvironments...))
				terraformProcessStep.Channels = sliceutil.NilIfEmpty(dependencies.GetResources("Channels", action.Channels...))
				terraformProcessStep.TenantTags = sliceutil.NilIfEmpty(c.Excluder.FilteredTenantTags(action.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets))
				terraformProcessStep.Condition = action.Condition
				terraformProcessStep.GitDependencies = c.OctopusActionProcessor.ConvertGitDependenciesV2(action.GitDependencies, dependencies)
				terraformProcessStep.IsDisabled = boolutil.NilIfFalse(action.IsDisabled)
				terraformProcessStep.IsRequired = boolutil.NilIfFalse(action.IsRequired)
				terraformProcessStep.Notes = action.Notes
				terraformProcessStep.Slug = action.Slug

				// Add the step to the list of steps
				terraformProcessSteps = append(terraformProcessSteps, terraformProcessStepBlock{
					Step:          &terraformProcessStep,
					OctopusStep:   &s,
					OctopusAction: &action,
					Block:         gohcl.EncodeAsBlock(terraformProcessStep, "resource"),
				})
			} else {
				// This is the parent step
				terraformProcessSteps = append(terraformProcessSteps, terraformProcessStepBlock{
					Step:          &terraformProcessStep,
					OctopusStep:   &s,
					OctopusAction: nil, // This indicates that this is a parent step
					Block:         gohcl.EncodeAsBlock(terraformProcessStep, "resource"),
				})

				for _, action := range s.Actions {
					terraformProcessStepChild := terraform.TerraformProcessStep{
						Type:                 "octopusdeploy_process_step",
						Name:                 "process_step_child_" + sanitizer.SanitizeNamePointer(s.Name),
						Id:                   nil,
						ResourceName:         strutil.EmptyIfNil(s.Name),
						ResourceType:         "",
						ProcessId:            "${octopusdeploy_process." + terraformProcessResource.Name + ".id}",
						Channels:             sliceutil.NilIfEmpty(dependencies.GetResources("Channels", action.Channels...)),
						Condition:            s.Condition,
						Container:            c.OctopusActionProcessor.ConvertContainerV2(action.Container, dependencies),
						Environments:         sliceutil.NilIfEmpty(dependencies.GetResources("Environments", action.Environments...)),
						ExcludedEnvironments: sliceutil.NilIfEmpty(dependencies.GetResources("Environments", action.ExcludedEnvironments...)),
						ExecutionProperties:  nil,
						GitDependencies:      c.OctopusActionProcessor.ConvertGitDependenciesV2(action.GitDependencies, dependencies),
						IsDisabled:           boolutil.NilIfFalse(action.IsDisabled),
						IsRequired:           boolutil.NilIfFalse(action.IsRequired),
						Notes:                action.Notes,
						Packages:             nil,
						PrimaryPackage:       nil,
						Slug:                 action.Slug,
						SpaceId:              nil,
						TenantTags:           sliceutil.NilIfEmpty(c.Excluder.FilteredTenantTags(action.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets)),
						WorkerPoolId:         nil,
						WorkerPoolVariable:   strutil.NilIfEmptyPointer(action.WorkerPoolVariable),
						StartTrigger:         s.StartTrigger,
						Properties:           nil,
						PackageRequirement:   s.PackageRequirement,
					}

					if err := c.assignWorkerPool(&terraformProcessStepChild, &action, file, dependencies); err != nil {
						return "", err
					}

					// This is the child step
					terraformProcessStepsChildren = append(terraformProcessSteps, terraformProcessStepBlock{
						Step:          &terraformProcessStepChild,
						OctopusStep:   nil, // This indicates that this is a child step
						OctopusAction: &action,
						Block:         gohcl.EncodeAsBlock(terraformProcessStep, "resource"),
					})
				}
			}
		}

		// The steps are captured in the TerraformProcessStepsOrder resource.
		terraformProcessStepsOrder := terraform.TerraformProcessStepsOrder{
			Type:      "octopusdeploy_process_steps_order",
			Name:      resourceName,
			Id:        nil,
			ProcessId: "${octopusdeploy_process." + terraformProcessResource.Name + ".id}",
			Steps: lo.Map(terraformProcessSteps, func(item terraformProcessStepBlock, index int) string {
				return "${octopusdeploy_process_step." + item.Step.Name + ".id}"
			}),
		}

		if stateless {
			// only create the deployment process, step order, and steps if the project was created
			terraformProcessResource.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", project.Id))
			terraformProcessStepsOrder.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", project.Id))
			lo.ForEach(terraformProcessSteps, func(item terraformProcessStepBlock, index int) {
				item.Step.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", project.Id))
			})
			lo.ForEach(terraformProcessStepsChildren, func(item terraformProcessStepBlock, index int) {
				item.Step.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", project.Id))
			})
		}

		terraformProcessResourceBlock := gohcl.EncodeAsBlock(terraformProcessResource, "resource")

		allTenantTags := lo.FlatMap(resource.Steps, func(item octopus.Step, index int) []string {
			return lo.FlatMap(item.Actions, func(item octopus.Action, index int) []string {
				if item.TenantTags != nil {
					return item.TenantTags
				}
				return []string{}
			})
		})

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, allTenantTags, c.TagSetConverter, terraformProcessResourceBlock, dependencies, recursive)

		if err != nil {
			return "", err
		}

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(terraformProcessResourceBlock)
		}

		file.Body().AppendBlock(terraformProcessResourceBlock)

		// Write the steps order
		terraformProcessStepsOrderBlock := gohcl.EncodeAsBlock(terraformProcessStepsOrder, "resource")
		file.Body().AppendBlock(terraformProcessStepsOrderBlock)

		// Write the steps
		lo.ForEach(terraformProcessSteps, func(item terraformProcessStepBlock, index int) {
			// execution properties are those that define how a single step or child step runs.
			// These properties are not used for the parent of a child step.
			if item.OctopusAction != nil {
				c.assignProperties("execution_properties", item.Block, project, item.OctopusAction.Properties, item.OctopusAction, file, dependencies)
			}
			// properties are used by a single step or a parent step.
			// These properties are not used for child steps.
			if item.OctopusStep != nil {
				c.assignProperties("properties", item.Block, project, maputil.ToStringAnyMap(item.OctopusStep.Properties), item.OctopusStep, file, dependencies)
			}
			file.Body().AppendBlock(item.Block)
		})

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c *DeploymentProcessConverterV2) assignProperties(propertyName string, block *hclwrite.Block, project octopus.Project, properties map[string]any, action octopus.NamedResource, file *hclwrite.File, dependencies *data.ResourceDetailsCollection) {
	if action == nil {
		return
	}

	sanitizedProperties, variables := steps.MapSanitizer{
		DummySecretGenerator:      c.DummySecretGenerator,
		DummySecretVariableValues: c.DummySecretVariableValues,
	}.SanitizeMap(project, action, properties, dependencies)
	sanitizedProperties = c.OctopusActionProcessor.EscapeDollars(sanitizedProperties)
	sanitizedProperties = c.OctopusActionProcessor.EscapePercents(sanitizedProperties)
	sanitizedProperties = c.OctopusActionProcessor.ReplaceStepTemplateVersion(dependencies, sanitizedProperties)
	sanitizedProperties = c.OctopusActionProcessor.ReplaceIds(c.ExperimentalEnableStepTemplates, sanitizedProperties, dependencies)
	sanitizedProperties = c.OctopusActionProcessor.RemoveUnnecessaryActionFields(sanitizedProperties)
	sanitizedProperties = c.OctopusActionProcessor.DetachStepTemplates(sanitizedProperties)
	sanitizedProperties = c.OctopusActionProcessor.LimitPropertyLength(c.LimitAttributeLength, true, sanitizedProperties)

	hcl.WriteStepProperties(propertyName, block, sanitizedProperties)

	for _, propertyVariables := range variables {
		propertyVariablesBlock := gohcl.EncodeAsBlock(propertyVariables, "variable")
		hcl.WriteUnquotedAttribute(propertyVariablesBlock, "type", "string")
		file.Body().AppendBlock(propertyVariablesBlock)
	}
}

func (c *DeploymentProcessConverterV2) assignPrimaryPackage(projectName string, terraformProcessStep *terraform.TerraformProcessStep, action *octopus.Action, file *hclwrite.File, dependencies *data.ResourceDetailsCollection) {
	primaryPackage, packageIdVariable := c.getPrimaryPackage(projectName, action, dependencies)

	if primaryPackage != nil {
		terraformProcessStep.PrimaryPackage = primaryPackage
		c.writeVariableToFile(file, packageIdVariable)
	}
}

func (c *DeploymentProcessConverterV2) assignReferencePackage(projectName string, terraformProcessStep *terraform.TerraformProcessStep, action *octopus.Action, file *hclwrite.File, dependencies *data.ResourceDetailsCollection) {
	referencePackages, referencePackageIdVariables := c.getPackages(projectName, action, dependencies)
	terraformProcessStep.Packages = referencePackages

	for _, variable := range referencePackageIdVariables {
		c.writeVariableToFile(file, variable)
	}
}

func (c *DeploymentProcessConverterV2) assignWorkerPool(terraformProcessStep *terraform.TerraformProcessStep, action *octopus.Action, file *hclwrite.File, dependencies *data.ResourceDetailsCollection) error {
	workerPoolId, err := c.WorkerPoolProcessor.ResolveWorkerPoolId(action.WorkerPoolId)

	if err != nil {
		return err
	}

	workerPool := ""
	if len(workerPoolId) != 0 {
		workerPool = dependencies.GetResource("WorkerPools", workerPoolId)
	}

	terraformProcessStep.WorkerPoolId = &workerPool

	return nil
}

// getPrimaryPackage returns the details of the primary package and an optional variable used to reference the package ID.
func (c *DeploymentProcessConverterV2) getPrimaryPackage(projectName string, action *octopus.Action, dependencies *data.ResourceDetailsCollection) (*terraform.TerraformProcessStepPackage, *terraform.TerraformVariable) {
	var packageIdVariable *terraform.TerraformVariable = nil

	for _, p := range action.Packages {
		packageId := strutil.EmptyIfNil(p.PackageId)

		var variableReference string

		// packages can be project IDs when they are defined in a "Deploy a release" step
		if regexes.ProjectsRegex.MatchString(packageId) && strutil.EmptyIfNil(action.ActionType) == "Octopus.DeployRelease" {
			variableReference = dependencies.GetResource("Projects", packageId)
		} else {
			packageIdVariable = c.getPackageIdVariable(packageId, projectName, strutil.EmptyIfNil(action.Name), strutil.EmptyIfNil(p.Name))
			variableReference = "${var." + packageIdVariable.Name + "}"
		}

		// Don't look up a feed id that is a variable reference
		feedId := p.FeedId
		if regexes.FeedRegex.MatchString(strutil.EmptyIfNil(feedId)) {
			feedId = dependencies.GetResourcePointer("Feeds", p.FeedId)
		}

		if strutil.NilIfEmptyPointer(p.Name) == nil {
			return &terraform.TerraformProcessStepPackage{
				Id:                  nil,
				PackageId:           variableReference,
				AcquisitionLocation: p.AcquisitionLocation,
				FeedId:              feedId,
				Properties:          maputil.NilIfEmptyMap(c.OctopusActionProcessor.ReplaceIds(c.ExperimentalEnableStepTemplates, p.Properties, dependencies)),
			}, packageIdVariable
		}
	}

	return nil, nil
}

// getPackages returns the details of the reference packages and an optional variables used to reference the package ID.
func (c *DeploymentProcessConverterV2) getPackages(projectName string, action *octopus.Action, dependencies *data.ResourceDetailsCollection) (*map[string]terraform.TerraformProcessStepPackage, []*terraform.TerraformVariable) {
	packages := map[string]terraform.TerraformProcessStepPackage{}
	packageIdVariables := []*terraform.TerraformVariable{}

	for _, p := range action.Packages {
		packageId := strutil.EmptyIfNil(p.PackageId)

		var variableReference string

		// packages can be project IDs when they are defined in a "Deploy a release" step
		if regexes.ProjectsRegex.MatchString(packageId) && strutil.EmptyIfNil(action.ActionType) == "Octopus.DeployRelease" {
			variableReference = dependencies.GetResource("Projects", packageId)
		} else {
			packageIdVariable := c.getPackageIdVariable(packageId, projectName, strutil.EmptyIfNil(action.Name), strutil.EmptyIfNil(p.Name))
			variableReference = "${var." + packageIdVariable.Name + "}"
			packageIdVariables = append(packageIdVariables, packageIdVariable)
		}

		// Don't look up a feed id that is a variable reference
		feedId := p.FeedId
		if regexes.FeedRegex.MatchString(strutil.EmptyIfNil(feedId)) {
			feedId = dependencies.GetResourcePointer("Feeds", p.FeedId)
		}

		if strutil.NilIfEmptyPointer(p.Name) != nil {
			packages[strutil.EmptyIfNil(p.Name)] = terraform.TerraformProcessStepPackage{
				Id:                  nil,
				PackageId:           variableReference,
				AcquisitionLocation: p.AcquisitionLocation,
				FeedId:              feedId,
				Properties:          maputil.NilIfEmptyMap(c.OctopusActionProcessor.ReplaceIds(c.ExperimentalEnableStepTemplates, p.Properties, dependencies)),
			}
		}
	}

	if len(packages) == 0 {
		return nil, nil
	}

	return &packages, packageIdVariables
}

func (c *DeploymentProcessConverterV2) GetResourceType() string {
	return "DeploymentProcesses"
}

func (c *DeploymentProcessConverterV2) exportDependencies(recursive bool, lookup bool, stateless bool, resource octopus.DeploymentProcess, dependencies *data.ResourceDetailsCollection) error {
	// Export linked accounts
	err := c.OctopusActionProcessor.ExportAccounts(recursive, lookup, stateless, resource.Steps, dependencies)
	if err != nil {
		return err
	}

	// Export linked feeds
	err = c.OctopusActionProcessor.ExportFeeds(recursive, lookup, stateless, resource.Steps, dependencies)
	if err != nil {
		return err
	}

	// Export linked worker pools
	err = c.OctopusActionProcessor.ExportWorkerPools(recursive, lookup, stateless, resource.Steps, dependencies)
	if err != nil {
		return err
	}

	// Export linked environments
	err = c.OctopusActionProcessor.ExportEnvironments(recursive, lookup, stateless, resource.Steps, dependencies)
	if err != nil {
		return err
	}

	// Export step templates
	err = c.OctopusActionProcessor.ExportStepTemplates(recursive, lookup, stateless, resource.Steps, dependencies)
	if err != nil {
		return err
	}

	// Export git credentials
	err = c.OctopusActionProcessor.ExportGitCredentials(recursive, lookup, stateless, resource.Steps, dependencies)
	if err != nil {
		return err
	}

	// Export projects, typically referenced in a "Deploy a release" step
	err = c.OctopusActionProcessor.ExportProjects(recursive, lookup, stateless, resource.Steps, dependencies)
	if err != nil {
		return err
	}

	return nil
}

func (c *DeploymentProcessConverterV2) getPackageIdVariable(defaultValue string, projectName string, stepName string, packageName string) *terraform.TerraformVariable {
	sanitizedProjectName := sanitizer.SanitizeName(projectName)
	sanitizedPackageName := sanitizer.SanitizeName(packageName)
	sanitizedStepName := sanitizer.SanitizeName(stepName)

	variableName := ""

	if packageName == "" {
		variableName = "project_" + sanitizedProjectName + "_step_" + sanitizedStepName + "_packageid"
	} else {
		variableName = "project_" + sanitizedProjectName + "_step_" + sanitizedStepName + "_package_" + sanitizedPackageName + "_packageid"
	}

	return &terraform.TerraformVariable{
		Name:        variableName,
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The package ID for the package named " + packageName + " from step " + stepName + " in project " + projectName,
		Default:     &defaultValue,
	}
}

func (c *DeploymentProcessConverterV2) writeVariableToFile(file *hclwrite.File, variable *terraform.TerraformVariable) {
	if variable == nil {
		return
	}

	block := gohcl.EncodeAsBlock(variable, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)
}
