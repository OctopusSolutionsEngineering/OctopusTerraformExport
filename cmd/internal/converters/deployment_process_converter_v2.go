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
	resourceName := c.generateProcessName(&project)

	err := c.exportDependencies(recursive, lookup, stateless, resource, dependencies)

	if err != nil {
		return err
	}

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_process." + resourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {
		file := hclwrite.NewEmptyFile()

		terraformProcessResource := terraform.TerraformProcess{
			Type:      "octopusdeploy_process",
			Name:      resourceName,
			Id:        nil,
			SpaceId:   nil,
			ProjectId: strutil.StrPointer(dependencies.GetResource("Projects", resource.ProjectId)),
			RunbookId: nil,
		}

		if stateless {
			// only create the deployment process, step order, and steps if the project was created
			terraformProcessResource.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", project.Id))
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

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	// Get all the valid steps
	validSteps := FilterSteps(
		resource.Steps,
		c.IgnoreInvalidExcludeExcept,
		c.Excluder,
		c.ExcludeAllSteps,
		c.ExcludeSteps,
		c.ExcludeStepsRegex,
		c.ExcludeStepsExcept)

	for _, step := range validSteps {
		parentStep := len(step.Actions) > 1

		// Every step is either standalone or a parent step with child steps.
		c.generateSteps(stateless, &resource, &project, &step, dependencies)

		if parentStep {
			// Steps that have children create a new child step from all the actions.
			for _, action := range step.Actions {
				c.generateChildSteps(stateless, &resource, &project, &action, dependencies)
			}

			// The child steps are captured in the TerraformProcessChildStepsOrder resource.
			c.generateChildStepOrder(stateless, &resource, &project, &step, dependencies)
		}
	}

	// The steps are captured in the TerraformProcessStepsOrder resource.
	c.generateStepOrder(stateless, &resource, &project, validSteps, dependencies)

	return nil
}

func (c *DeploymentProcessConverterV2) generateChildStepOrder(stateless bool, resource *octopus.DeploymentProcess, project *octopus.Project, step *octopus.Step, dependencies *data.ResourceDetailsCollection) {
	resourceName := c.generateChildStepOrderName(project, step)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.ParentId = project.Id
	thisResource.Id = project.Id + "/" + resource.Id + "/" + strutil.EmptyIfNil(step.Id)
	thisResource.ResourceType = "DeploymentProcesses/StepOrder"
	thisResource.Lookup = "${octopusdeploy_process_steps_order." + resourceName + ".id}"
	thisResource.Dependency = "${octopusdeploy_process_steps_order." + resourceName + "}"
	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		terraformProcessChildStepsOrder := terraform.TerraformProcessChildStepsOrder{
			Type:      "octopusdeploy_process_child_steps_order",
			Name:      resourceName,
			Id:        nil,
			ProcessId: "${octopusdeploy_process." + c.generateProcessName(project) + ".id}",
			ParentId:  "${octopusdeploy_process_step." + c.generateStepName(project, step) + ".id}",
			Steps: lo.Map(step.Actions, func(item octopus.Action, index int) string {
				return "${octopusdeploy_process_child_step." + c.generateChildStepName(project, &item) + ".id}"
			}),
		}

		if stateless {
			// only create the deployment process, step order, and steps if the project was created
			terraformProcessChildStepsOrder.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", project.Id))
		}

		block := gohcl.EncodeAsBlock(terraformProcessChildStepsOrder, "resource")

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
}

func (c *DeploymentProcessConverterV2) generateStepOrder(stateless bool, resource *octopus.DeploymentProcess, project *octopus.Project, steps []octopus.Step, dependencies *data.ResourceDetailsCollection) {
	resourceName := c.generateStepOrderName(project)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.ParentId = project.Id
	thisResource.Id = resource.Id
	thisResource.ResourceType = "DeploymentProcesses/StepOrder"
	thisResource.Lookup = "${octopusdeploy_process_steps_order." + resourceName + ".id}"
	thisResource.Dependency = "${octopusdeploy_process_steps_order." + resourceName + "}"
	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		terraformProcessStepsOrder := terraform.TerraformProcessStepsOrder{
			Type:      "octopusdeploy_process_steps_order",
			Name:      resourceName,
			Id:        nil,
			ProcessId: "${octopusdeploy_process." + c.generateProcessName(project) + ".id}",
			Steps: lo.Map(steps, func(item octopus.Step, index int) string {
				return "${octopusdeploy_process_step." + c.generateStepName(project, &item) + ".id}"
			}),
		}

		if stateless {
			// only create the deployment process, step order, and steps if the project was created
			terraformProcessStepsOrder.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", project.Id))
		}

		block := gohcl.EncodeAsBlock(terraformProcessStepsOrder, "resource")

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
}

func (c *DeploymentProcessConverterV2) generateChildSteps(stateless bool, resource *octopus.DeploymentProcess, project *octopus.Project, action *octopus.Action, dependencies *data.ResourceDetailsCollection) {
	resourceName := c.generateChildStepName(project, action)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.ParentId = project.Id
	thisResource.Id = project.Id + "/" + resource.Id + "/" + action.Id
	thisResource.ResourceType = "DeploymentProcesses/Steps"
	thisResource.Lookup = "${octopusdeploy_process_child_step." + resourceName + ".id}"
	thisResource.Dependency = "${octopusdeploy_process_child_step." + resourceName + "}"
	thisResource.ToHcl = func() (string, error) {
		file := hclwrite.NewEmptyFile()

		terraformProcessStepChild := terraform.TerraformProcessStep{
			Type:                 "octopusdeploy_process_child_step",
			Name:                 resourceName,
			Id:                   nil, // Read only
			ResourceName:         strutil.EmptyIfNil(action.Name),
			ResourceType:         strutil.EmptyIfNil(action.ActionType),
			ProcessId:            "${octopusdeploy_process." + c.generateProcessName(project) + ".id}",
			Channels:             sliceutil.NilIfEmpty(dependencies.GetResources("Channels", action.Channels...)),
			Condition:            action.Condition,
			Container:            c.OctopusActionProcessor.ConvertContainerV2(action.Container, dependencies),
			Environments:         sliceutil.NilIfEmpty(dependencies.GetResources("Environments", action.Environments...)),
			ExcludedEnvironments: sliceutil.NilIfEmpty(dependencies.GetResources("Environments", action.ExcludedEnvironments...)),
			ExecutionProperties:  nil, // This is assigned by assignProperties()
			GitDependencies:      c.OctopusActionProcessor.ConvertGitDependenciesV2(action.GitDependencies, dependencies),
			IsDisabled:           boolutil.NilIfFalse(action.IsDisabled),
			IsRequired:           boolutil.NilIfFalse(action.IsRequired),
			Notes:                action.Notes,
			Packages:             nil, // This is assigned by assignPrimaryPackage()
			PrimaryPackage:       nil, // This is assigned by assignPrimaryPackage()
			Slug:                 action.Slug,
			SpaceId:              nil,
			TenantTags:           sliceutil.NilIfEmpty(c.Excluder.FilteredTenantTags(action.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets)),
			WorkerPoolId:         nil, // This is assigned by assignWorkerPool()
			WorkerPoolVariable:   strutil.NilIfEmptyPointer(action.WorkerPoolVariable),
			StartTrigger:         nil, // This is not defined on a child step
			Properties:           nil, // These properties are not used for child steps
			PackageRequirement:   nil, // This is not defined on a child step
		}

		if stateless {
			// only create the deployment process, step order, and steps if the project was created
			terraformProcessStepChild.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", project.Id))
		}

		c.assignPrimaryPackage(project.Name, &terraformProcessStepChild, action, file, dependencies)
		c.assignReferencePackage(project.Name, &terraformProcessStepChild, action, file, dependencies)
		if err := c.assignWorkerPool(&terraformProcessStepChild, action, file, dependencies); err != nil {
			return "", err
		}

		block := gohcl.EncodeAsBlock(terraformProcessStepChild, "resource")

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		c.assignProperties("execution_properties", block, project, action.Properties, action, file, dependencies)

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
}

func (c *DeploymentProcessConverterV2) generateSteps(stateless bool, resource *octopus.DeploymentProcess, project *octopus.Project, step *octopus.Step, dependencies *data.ResourceDetailsCollection) {
	resourceName := c.generateStepName(project, step)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.ParentId = project.Id
	thisResource.Id = project.Id + "/" + resource.Id + "/" + strutil.EmptyIfNil(step.Id)
	thisResource.ResourceType = "DeploymentProcesses/Steps"
	thisResource.Lookup = "${octopusdeploy_process_step." + resourceName + ".id}"
	thisResource.Dependency = "${octopusdeploy_process_step." + resourceName + "}"
	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		terraformProcessStep := terraform.TerraformProcessStep{
			Type:                 "octopusdeploy_process_step",
			Name:                 resourceName,
			Id:                   nil,
			ResourceName:         strutil.EmptyIfNil(step.Name),
			ResourceType:         "",
			ProcessId:            "${octopusdeploy_process." + c.generateProcessName(project) + ".id}",
			Channels:             nil,
			Condition:            step.Condition,
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
			StartTrigger:         step.StartTrigger,
			Properties:           nil,
			PackageRequirement:   step.PackageRequirement,
		}

		standaloneStep := len(step.Actions) == 1

		// We build the output differently for a step with a single action (represented as a typical step in the UI)
		// and a step with multiple actions (represented as a parent step with child steps in the UI).
		if standaloneStep {
			action := step.Actions[0]

			// The step type is the type of the first action.
			terraformProcessStep.ResourceType = strutil.EmptyIfNil(action.ActionType)

			c.assignPrimaryPackage(project.Name, &terraformProcessStep, &action, file, dependencies)
			c.assignReferencePackage(project.Name, &terraformProcessStep, &action, file, dependencies)
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
			terraformProcessStep.ResourceType = strutil.EmptyIfNil(action.ActionType)
		}

		if stateless {
			// only create the deployment process, step order, and steps if the project was created
			terraformProcessStep.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", project.Id))
		}

		block := gohcl.EncodeAsBlock(terraformProcessStep, "resource")

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		c.assignProperties("properties", block, project, maputil.ToStringAnyMap(step.Properties), step, file, dependencies)

		if standaloneStep {
			c.assignProperties("execution_properties", block, project, step.Actions[0].Properties, &step.Actions[0], file, dependencies)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
}

func (c *DeploymentProcessConverterV2) generateProcessName(project *octopus.Project) string {
	return "process_" + sanitizer.SanitizeName(project.Name)
}

func (c *DeploymentProcessConverterV2) generateStepOrderName(project *octopus.Project) string {
	return "process_step_order_" + sanitizer.SanitizeName(project.Name)
}

func (c *DeploymentProcessConverterV2) generateChildStepOrderName(project *octopus.Project, named octopus.NamedResource) string {
	return "process_child_step_order_" + sanitizer.SanitizeName(project.Name) + "_" + sanitizer.SanitizeName(named.GetName())
}

func (c *DeploymentProcessConverterV2) generateStepName(project *octopus.Project, named octopus.NamedResource) string {
	return "process_step_" + sanitizer.SanitizeName(project.Name) + "_" + sanitizer.SanitizeName(named.GetName())
}

func (c *DeploymentProcessConverterV2) generateChildStepName(project *octopus.Project, named octopus.NamedResource) string {
	return "process_child_step_" + sanitizer.SanitizeName(project.Name) + "_" + sanitizer.SanitizeName(named.GetName())
}

func (c *DeploymentProcessConverterV2) assignProperties(propertyName string, block *hclwrite.Block, project *octopus.Project, properties map[string]any, action octopus.NamedResource, file *hclwrite.File, dependencies *data.ResourceDetailsCollection) {
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
	// Worker pool variable takes precedence over worker pool ID
	if action.WorkerPoolVariable != nil {
		return nil
	}

	workerPoolId, err := c.WorkerPoolProcessor.ResolveWorkerPoolId(action.WorkerPoolId)

	if err != nil {
		return err
	}

	workerPool := ""
	if len(workerPoolId) != 0 {
		workerPool = dependencies.GetResource("WorkerPools", workerPoolId)
	}

	terraformProcessStep.WorkerPoolId = strutil.NilIfEmpty(workerPool)

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
