package converters

import (
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
)

const octopusdeployProcessResourceType = "octopusdeploy_process"
const octopusdeployProcessStepResourceType = "octopusdeploy_process_step"
const octopusdeployProcessTemplateStepResourceType = "octopusdeploy_process_templated_step"
const octopusdeployProcessChildStepResourceType = "octopusdeploy_process_child_step"
const octopusdeployProcessStepsOrderResourceType = "octopusdeploy_process_steps_order"
const octopusdeployProcessTemplatedStepsOrderResourceType = "octopusdeploy_process_templated_child_step"
const octopusdeployProcessStepsOrder = "octopusdeploy_process_steps_order"
const octopusdeployProcessChildStepsOrder = "octopusdeploy_process_child_steps_order"

type DeploymentProcessConverterBase struct {
	ResourceType                    string
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
	DetachProjectTemplates          bool
	GenerateImportScripts           bool
}

func (c *DeploymentProcessConverterBase) SetActionProcessor(actionProcessor *OctopusActionProcessor) {
	c.OctopusActionProcessor = actionProcessor
}

func (c *DeploymentProcessConverterBase) getParentName(parent octopus.NameIdParentResource, owner octopus.NameIdParentResource) string {
	if parent != nil {
		return parent.GetName()
	}

	return owner.GetName()
}

func (c *DeploymentProcessConverterBase) toHcl(deploymentProcess octopus.OctopusProcess, parentProjectOrNil octopus.NameIdParentResource, projectOrRunbook octopus.NameIdParentResource, recursive bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	resourceName := c.generateProcessName(parentProjectOrNil, projectOrRunbook)
	projectResourceName := "project_" + sanitizer.SanitizeName(c.getParentName(parentProjectOrNil, projectOrRunbook))

	err := c.exportDependencies(recursive, lookup, stateless, deploymentProcess, dependencies)

	if err != nil {
		return err
	}

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = deploymentProcess.GetId()
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Dependency = "${" + octopusdeployProcessResourceType + "." + resourceName + "}"

	if stateless {
		// There is no way to look up an existing deployment process. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the runbook anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + projectResourceName + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProcessResourceType + "." + resourceName + "[0].id}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProcessResourceType + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {
		file := hclwrite.NewEmptyFile()

		terraformProcessResource := terraform.TerraformProcess{
			Type:      octopusdeployProcessResourceType,
			Name:      resourceName,
			Id:        nil,
			SpaceId:   nil,
			ProjectId: strutil.StrPointer(dependencies.GetResource("Projects", projectOrRunbook.GetUltimateParent())),
			RunbookId: nil,
		}

		// If there is a parent resource, we are working with a runbook process.
		if parentProjectOrNil != nil {
			terraformProcessResource.RunbookId = strutil.NilIfEmpty(dependencies.GetResource("Runbooks", projectOrRunbook.GetId()))
		}

		if stateless {
			// only create the deployment process, step order, and steps if the project was created
			terraformProcessResource.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", projectOrRunbook.GetUltimateParent()))
		}

		terraformProcessResourceBlock := gohcl.EncodeAsBlock(terraformProcessResource, "resource")

		allTenantTags := lo.FlatMap(deploymentProcess.GetSteps(), func(item octopus.Step, index int) []string {
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
	validSteps := c.getValidSteps(deploymentProcess)

	for _, step := range validSteps {
		parentStep := len(step.Actions) > 1

		// Every step is either standalone or a parent step with child steps.
		c.generateSteps(stateless, deploymentProcess, parentProjectOrNil, projectOrRunbook, &step, dependencies)
		c.generateTemplateSteps(stateless, deploymentProcess, parentProjectOrNil, projectOrRunbook, &step, dependencies)

		if parentStep {
			// Steps that have children create a new child step from all the actions after the first one.
			for _, action := range step.Actions[1:] {
				c.generateChildSteps(stateless, deploymentProcess, parentProjectOrNil, projectOrRunbook, &step, &action, dependencies)
				c.generateTemplateChildSteps(stateless, deploymentProcess, parentProjectOrNil, projectOrRunbook, &step, &action, dependencies)
			}

			// The child steps are captured in the TerraformProcessChildStepsOrder resource.
			c.generateChildStepOrder(stateless, deploymentProcess, parentProjectOrNil, projectOrRunbook, &step, dependencies)
		}
	}

	// The steps are captured in the TerraformProcessStepsOrder resource.
	c.generateStepOrder(stateless, deploymentProcess, parentProjectOrNil, projectOrRunbook, validSteps, dependencies)

	return nil
}

func (c *DeploymentProcessConverterBase) getValidSteps(resource octopus.OctopusProcess) []octopus.Step {
	return FilterSteps(
		resource.GetSteps(),
		c.IgnoreInvalidExcludeExcept,
		c.Excluder,
		c.ExcludeAllSteps,
		c.ExcludeSteps,
		c.ExcludeStepsRegex,
		c.ExcludeStepsExcept)
}

func (c *DeploymentProcessConverterBase) generateChildStepOrder(stateless bool, deploymentProcess octopus.OctopusProcess, parent octopus.NameIdParentResource, owner octopus.NameIdParentResource, step *octopus.Step, dependencies *data.ResourceDetailsCollection) {
	if len(step.Actions) < 2 {
		// This shouldn't happen, but if a step has no child actions, we don't create a child step order.
		return
	}

	resourceName := c.generateChildStepOrderName(parent, owner, step)
	projectResourceName := "project_" + sanitizer.SanitizeName(c.getParentName(parent, owner))

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.ParentId = owner.GetUltimateParent()
	thisResource.Id = c.getStepId(deploymentProcess, owner, step)
	thisResource.ResourceType = "DeploymentProcesses/StepOrder"
	thisResource.Dependency = "${" + octopusdeployProcessChildStepsOrder + "." + resourceName + "}"

	if stateless {
		// There is no way to look up an existing deployment process. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the runbook anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + projectResourceName + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProcessChildStepsOrder + "." + resourceName + "[0].id}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProcessChildStepsOrder + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		terraformProcessChildStepsOrder := terraform.TerraformProcessChildStepsOrder{
			Type:      octopusdeployProcessChildStepsOrder,
			Name:      resourceName,
			Id:        nil,
			ProcessId: dependencies.GetResource(c.GetResourceType(), deploymentProcess.GetId()),
			ParentId:  dependencies.GetResource("DeploymentProcesses/Steps", c.getStepId(deploymentProcess, owner, step)),
			// The first action is folded in the parent step, so we don't include it in the child steps.
			// The child steps are the second action on and onwards in the step.
			Children: lo.Map(step.Actions[1:], func(item octopus.Action, index int) string {
				return dependencies.GetResource("DeploymentProcesses/ChildSteps", owner.GetId()+"/"+deploymentProcess.GetId()+"/"+item.Id)
			}),
		}

		if stateless {
			// only create the deployment process, step order, and steps if the project was created
			terraformProcessChildStepsOrder.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", owner.GetUltimateParent()))
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

func (c *DeploymentProcessConverterBase) generateStepOrder(stateless bool, resource octopus.OctopusProcess, parent octopus.NameIdParentResource, owner octopus.NameIdParentResource, steps []octopus.Step, dependencies *data.ResourceDetailsCollection) {
	resourceName := c.generateStepOrderName(parent, owner)
	projectResourceName := "project_" + sanitizer.SanitizeName(c.getParentName(parent, owner))

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.ParentId = owner.GetUltimateParent()
	thisResource.Id = resource.GetId()
	thisResource.ResourceType = "DeploymentProcesses/StepOrder"
	thisResource.Dependency = "${" + octopusdeployProcessStepsOrderResourceType + "." + resourceName + "}"

	if stateless {
		// There is no way to look up an existing deployment process. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the runbook anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + projectResourceName + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProcessStepsOrderResourceType + "." + resourceName + "[0].id}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProcessStepsOrderResourceType + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		terraformProcessStepsOrder := terraform.TerraformProcessStepsOrder{
			Type:      octopusdeployProcessStepsOrderResourceType,
			Name:      resourceName,
			Id:        nil,
			ProcessId: dependencies.GetResource(c.GetResourceType(), resource.GetId()),
			Steps: lo.Map(steps, func(item octopus.Step, index int) string {
				return dependencies.GetResource("DeploymentProcesses/Steps", c.getStepId(resource, owner, &item))
			}),
		}

		if stateless {
			// only create the deployment process, step order, and steps if the project was created
			terraformProcessStepsOrder.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", owner.GetUltimateParent()))
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

func (c *DeploymentProcessConverterBase) generateChildSteps(stateless bool, resource octopus.OctopusProcess, parent octopus.NameIdParentResource, owner octopus.NameIdParentResource, step *octopus.Step, action *octopus.Action, dependencies *data.ResourceDetailsCollection) {
	if _, ok := action.Properties["Octopus.Action.Template.Id"]; ok && !c.DetachProjectTemplates {
		// This is a templated step, so we don't generate a resource for it.
		return
	}

	resourceName := c.generateChildStepName(parent, owner, action)
	projectResourceName := "project_" + sanitizer.SanitizeName(c.getParentName(parent, owner))

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.ParentId = owner.GetUltimateParent()
	thisResource.Id = owner.GetId() + "/" + resource.GetId() + "/" + action.Id
	thisResource.ResourceType = "DeploymentProcesses/ChildSteps"
	thisResource.Dependency = "${" + octopusdeployProcessChildStepResourceType + "." + resourceName + "}"

	if stateless {
		// There is no way to look up an existing deployment process. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the runbook anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + projectResourceName + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProcessChildStepResourceType + "." + resourceName + "[0].id}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProcessChildStepResourceType + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {
		file := hclwrite.NewEmptyFile()

		terraformProcessStepChild := terraform.TerraformProcessStep{
			Type:                 octopusdeployProcessChildStepResourceType,
			Name:                 resourceName,
			Id:                   nil, // Read only
			ResourceName:         strutil.EmptyIfNil(action.Name),
			ResourceType:         strutil.EmptyIfNil(action.ActionType),
			ProcessId:            dependencies.GetResource(c.GetResourceType(), resource.GetId()),
			ParentId:             strutil.NilIfEmpty(dependencies.GetResource("DeploymentProcesses/Steps", c.getStepId(resource, owner, step))),
			Channels:             sliceutil.NilIfEmpty(dependencies.GetResources("Channels", action.Channels...)),
			Condition:            action.Condition,
			Container:            c.OctopusActionProcessor.ConvertContainer(action.Container, dependencies),
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
			terraformProcessStepChild.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", owner.GetUltimateParent()))
		}

		c.assignPrimaryPackage(owner.GetName(), &terraformProcessStepChild, action, file, dependencies)
		c.assignReferencePackage(owner.GetName(), &terraformProcessStepChild, action, file, dependencies)
		if err := c.assignWorkerPool(&terraformProcessStepChild, action, file, dependencies); err != nil {
			return "", err
		}

		block := gohcl.EncodeAsBlock(terraformProcessStepChild, "resource")

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		c.assignProperties("execution_properties", block, owner, action.Properties, []string{}, action, file, dependencies)

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
}

func (c *DeploymentProcessConverterBase) generateTemplateChildSteps(stateless bool, resource octopus.OctopusProcess, parent octopus.NameIdParentResource, owner octopus.NameIdParentResource, step *octopus.Step, action *octopus.Action, dependencies *data.ResourceDetailsCollection) {
	templateId, ok := action.Properties["Octopus.Action.Template.Id"]

	if !ok || c.DetachProjectTemplates {
		// This is a templated step, so we don't generate a resource for it.
		return
	}

	resourceName := c.generateChildStepName(parent, owner, action)
	projectResourceName := "project_" + sanitizer.SanitizeName(c.getParentName(parent, owner))

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.ParentId = owner.GetUltimateParent()
	thisResource.Id = owner.GetId() + "/" + resource.GetId() + "/" + action.Id
	thisResource.ResourceType = "DeploymentProcesses/ChildSteps"
	thisResource.Dependency = "${" + octopusdeployProcessTemplatedStepsOrderResourceType + "." + resourceName + "}"

	if stateless {
		// There is no way to look up an existing deployment process. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the runbook anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + projectResourceName + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProcessTemplatedStepsOrderResourceType + "." + resourceName + "[0].id}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProcessTemplatedStepsOrderResourceType + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {
		file := hclwrite.NewEmptyFile()

		// The native step template data source does not have the ability to look up the template ID and version.
		// So we just reference them as is. This will work when a project is recreated in the same space,
		// but will fail across spaces as the template IDs change.
		newTemplateId := step.Actions[0].Properties["Octopus.Action.Template.Id"]
		newTemplateVersion := step.Actions[0].Properties["Octopus.Action.Template.Version"]

		// If the experimental flag is enabled, we use a workaround to query the template ID and version
		// from the API.
		if c.ExperimentalEnableStepTemplates {
			newTemplateId = dependencies.GetResource("ActionTemplates", templateId.(string))
			newTemplateVersion = dependencies.GetResourceVersionLookup("ActionTemplates", templateId.(string))
		}

		terraformProcessStepChild := terraform.TerraformProcessTemplatedStep{
			Type:                 octopusdeployProcessTemplatedStepsOrderResourceType,
			Name:                 resourceName,
			TemplateId:           newTemplateId.(string),
			TemplateVersion:      newTemplateVersion.(string),
			ResourceName:         strutil.EmptyIfNil(action.Name),
			ProcessId:            dependencies.GetResource(c.GetResourceType(), resource.GetId()),
			ParentId:             strutil.NilIfEmpty(dependencies.GetResource("DeploymentProcesses/Steps", c.getStepId(resource, owner, step))),
			Channels:             sliceutil.NilIfEmpty(dependencies.GetResources("Channels", action.Channels...)),
			Condition:            action.Condition,
			Container:            c.OctopusActionProcessor.ConvertContainer(action.Container, dependencies),
			Environments:         sliceutil.NilIfEmpty(dependencies.GetResources("Environments", action.Environments...)),
			ExcludedEnvironments: sliceutil.NilIfEmpty(dependencies.GetResources("Environments", action.ExcludedEnvironments...)),
			ExecutionProperties:  nil, // This is assigned by assignProperties()
			IsDisabled:           boolutil.NilIfFalse(action.IsDisabled),
			IsRequired:           boolutil.NilIfFalse(action.IsRequired),
			Notes:                action.Notes,
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
			terraformProcessStepChild.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", owner.GetUltimateParent()))
		}

		if err := c.assignWorkerPool(&terraformProcessStepChild, action, file, dependencies); err != nil {
			return "", err
		}

		block := gohcl.EncodeAsBlock(terraformProcessStepChild, "resource")

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		if parameters, err := c.getTemplateParameters(templateId.(string)); err != nil {
			return "", err
		} else {
			c.assignProperties("execution_properties", block, owner, action.Properties, parameters, action, file, dependencies)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
}

// getTemplateParameters retrieves the parameters of a step template by its ID.
func (c *DeploymentProcessConverterBase) getTemplateParameters(templateId string) ([]string, error) {
	template := octopus.StepTemplate{}
	_, err := c.Client.GetSpaceResourceById("ActionTemplates", templateId, &template)

	if err != nil {
		return nil, err
	}

	return lo.Map(template.Parameters, func(item octopus.StepTemplateParameters, index int) string {
		return item.Name
	}), nil
}

func (c *DeploymentProcessConverterBase) generateTemplateSteps(stateless bool, resource octopus.OctopusProcess, parent octopus.NameIdParentResource, owner octopus.NameIdParentResource, step *octopus.Step, dependencies *data.ResourceDetailsCollection) {
	// This should always be true, but we check it to avoid panics.
	hasChild := len(step.Actions) >= 1

	if !hasChild {
		return
	}

	templateId, ok := step.Actions[0].Properties["Octopus.Action.Template.Id"]
	if !ok || c.DetachProjectTemplates {
		// This is not a templated step, so we don't generate a resource for it.
		return
	}

	resourceName := c.generateStepName(parent, owner, step)
	projectResourceName := "project_" + sanitizer.SanitizeName(c.getParentName(parent, owner))

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.ParentId = owner.GetUltimateParent()
	thisResource.Id = c.getStepId(resource, owner, step)
	thisResource.ResourceType = "DeploymentProcesses/Steps"
	thisResource.Dependency = "${" + octopusdeployProcessTemplateStepResourceType + "." + resourceName + "}"

	if stateless {
		// There is no way to look up an existing deployment process. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the runbook anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + projectResourceName + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProcessTemplateStepResourceType + "." + resourceName + "[0].id}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProcessTemplateStepResourceType + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		// The native step template data source does not have the ability to look up the template ID and version.
		// So we just reference them as is. This will work when a project is recreated in the same space,
		// but will fail across spaces as the template IDs change.
		newTemplateId := step.Actions[0].Properties["Octopus.Action.Template.Id"]
		newTemplateVersion := step.Actions[0].Properties["Octopus.Action.Template.Version"]

		// If the experimental flag is enabled, we use a workaround to query the template ID and version
		// from the API.
		if c.ExperimentalEnableStepTemplates {
			newTemplateId = dependencies.GetResource("ActionTemplates", templateId.(string))
			newTemplateVersion = dependencies.GetResourceVersionLookup("ActionTemplates", templateId.(string))
		}

		terraformProcessStep := terraform.TerraformProcessTemplatedStep{
			Type:                 octopusdeployProcessTemplateStepResourceType,
			Name:                 resourceName,
			Count:                nil,
			ResourceName:         strutil.EmptyIfNil(step.Name),
			ParentId:             nil,
			ProcessId:            dependencies.GetResource(c.GetResourceType(), resource.GetId()),
			TemplateId:           newTemplateId.(string),
			TemplateVersion:      newTemplateVersion.(string),
			Channels:             nil,
			Condition:            step.Condition,
			Container:            nil,
			Environments:         nil,
			ExcludedEnvironments: nil,
			ExecutionProperties:  nil,
			IsDisabled:           nil,
			IsRequired:           nil,
			Notes:                nil,
			PackageRequirement:   step.PackageRequirement,
			Parameters:           nil,
			Properties:           nil,
			Slug:                 nil,
			SpaceId:              nil,
			StartTrigger:         step.StartTrigger,
			TenantTags:           nil,
			WorkerPoolId:         nil,
			WorkerPoolVariable:   nil,
		}

		// This should always be true, but we check it to avoid panics.
		hasChild := len(step.Actions) >= 1

		// We build the output differently for a step with a single action (represented as a typical step in the UI)
		// and a step with multiple actions (represented as a parent step with child steps in the UI).
		if hasChild {
			action := step.Actions[0]

			// The step type is the type of the first action.
			if err := c.assignWorkerPool(&terraformProcessStep, &action, file, dependencies); err != nil {
				return "", err
			}

			terraformProcessStep.Container = c.OctopusActionProcessor.ConvertContainer(action.Container, dependencies)
			terraformProcessStep.WorkerPoolVariable = strutil.NilIfEmptyPointer(action.WorkerPoolVariable)
			terraformProcessStep.Environments = sliceutil.NilIfEmpty(dependencies.GetResources("Environments", action.Environments...))
			terraformProcessStep.ExcludedEnvironments = sliceutil.NilIfEmpty(dependencies.GetResources("Environments", action.ExcludedEnvironments...))
			terraformProcessStep.Channels = sliceutil.NilIfEmpty(dependencies.GetResources("Channels", action.Channels...))
			terraformProcessStep.TenantTags = sliceutil.NilIfEmpty(c.Excluder.FilteredTenantTags(action.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets))
			terraformProcessStep.IsDisabled = boolutil.NilIfFalse(action.IsDisabled)
			terraformProcessStep.IsRequired = boolutil.NilIfFalse(action.IsRequired)
			terraformProcessStep.Notes = action.Notes
			terraformProcessStep.Slug = action.Slug
		}

		if stateless {
			// only create the deployment process, step order, and steps if the project was created
			terraformProcessStep.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", owner.GetUltimateParent()))
		}

		block := gohcl.EncodeAsBlock(terraformProcessStep, "resource")

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		c.assignProperties("properties", block, owner, maputil.ToStringAnyMap(step.Properties), []string{}, step, file, dependencies)

		if hasChild {

			template := octopus.StepTemplate{}
			_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), templateId.(string), &template)

			if err != nil {
				return "", err
			}

			if parameters, err := c.getTemplateParameters(templateId.(string)); err != nil {
				return "", err
			} else {
				c.assignProperties("execution_properties", block, owner, step.Actions[0].Properties, parameters, &step.Actions[0], file, dependencies)
			}
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
}

func (c *DeploymentProcessConverterBase) generateSteps(stateless bool, deploymentProcess octopus.OctopusProcess, parentProjectOrNil octopus.NameIdParentResource, projectOrRunbook octopus.NameIdParentResource, step *octopus.Step, dependencies *data.ResourceDetailsCollection) {
	// This should always be true, but we check it to avoid panics.
	hasChild := len(step.Actions) >= 1

	if !hasChild {
		return
	}

	// We process this step if it is not a template or if we are detaching project templates.
	if _, ok := step.Actions[0].Properties["Octopus.Action.Template.Id"]; ok && !c.DetachProjectTemplates {
		return
	}

	resourceName := c.generateStepName(parentProjectOrNil, projectOrRunbook, step)
	projectResourceName := "project_" + sanitizer.SanitizeName(c.getParentName(parentProjectOrNil, projectOrRunbook))

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.ParentId = projectOrRunbook.GetUltimateParent()
	thisResource.Id = c.getStepId(deploymentProcess, projectOrRunbook, step)
	thisResource.AlternateId = c.getActionId(deploymentProcess, projectOrRunbook, &step.Actions[0])
	thisResource.ResourceType = "DeploymentProcesses/Steps"
	thisResource.Dependency = "${" + octopusdeployProcessStepResourceType + "." + resourceName + "}"

	if stateless {
		// There is no way to look up an existing deployment process. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the runbook anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + projectResourceName + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProcessStepResourceType + "." + resourceName + "[0].id}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProcessStepResourceType + "." + resourceName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		terraformProcessStep := terraform.TerraformProcessStep{
			Type:                 octopusdeployProcessStepResourceType,
			Name:                 resourceName,
			Id:                   nil,
			ResourceName:         strutil.EmptyIfNil(step.Name),
			ResourceType:         "Dummy",
			ProcessId:            dependencies.GetResource(c.GetResourceType(), deploymentProcess.GetId()),
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

		// We build the output differently for a step with a single action (represented as a typical step in the UI)
		// and a step with multiple actions (represented as a parent step with child steps in the UI).
		if hasChild {
			action := step.Actions[0]

			// The step type is the type of the first action.
			terraformProcessStep.ResourceType = strutil.EmptyIfNil(action.ActionType)

			c.assignPrimaryPackage(projectOrRunbook.GetName(), &terraformProcessStep, &action, file, dependencies)
			c.assignReferencePackage(projectOrRunbook.GetName(), &terraformProcessStep, &action, file, dependencies)
			if err := c.assignWorkerPool(&terraformProcessStep, &action, file, dependencies); err != nil {
				return "", err
			}

			terraformProcessStep.Container = c.OctopusActionProcessor.ConvertContainer(action.Container, dependencies)
			terraformProcessStep.WorkerPoolVariable = strutil.NilIfEmptyPointer(action.WorkerPoolVariable)
			terraformProcessStep.Environments = sliceutil.NilIfEmpty(dependencies.GetResources("Environments", action.Environments...))
			terraformProcessStep.ExcludedEnvironments = sliceutil.NilIfEmpty(dependencies.GetResources("Environments", action.ExcludedEnvironments...))
			terraformProcessStep.Channels = sliceutil.NilIfEmpty(dependencies.GetResources("Channels", action.Channels...))
			terraformProcessStep.TenantTags = sliceutil.NilIfEmpty(c.Excluder.FilteredTenantTags(action.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets))
			terraformProcessStep.GitDependencies = c.OctopusActionProcessor.ConvertGitDependenciesV2(action.GitDependencies, dependencies)
			terraformProcessStep.IsDisabled = boolutil.NilIfFalse(action.IsDisabled)
			terraformProcessStep.IsRequired = boolutil.NilIfFalse(action.IsRequired)
			terraformProcessStep.Notes = action.Notes
			terraformProcessStep.Slug = action.Slug
			terraformProcessStep.ResourceType = strutil.EmptyIfNil(action.ActionType)
		}

		if stateless {
			// only create the deployment process, step order, and steps if the project was created
			terraformProcessStep.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", projectOrRunbook.GetUltimateParent()))
		}

		block := gohcl.EncodeAsBlock(terraformProcessStep, "resource")

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		c.assignProperties("properties", block, projectOrRunbook, maputil.ToStringAnyMap(step.Properties), []string{}, step, file, dependencies)

		if hasChild {
			c.assignProperties("execution_properties", block, projectOrRunbook, step.Actions[0].Properties, []string{}, &step.Actions[0], file, dependencies)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
}

func (c *DeploymentProcessConverterBase) generateProcessName(parent octopus.NameIdParentResource, owner octopus.NameIdParentResource) string {
	if parent != nil {
		return "process_" + sanitizer.SanitizeName(parent.GetName()) + "_" + sanitizer.SanitizeName(owner.GetName())
	}
	return "process_" + sanitizer.SanitizeName(owner.GetName())
}

func (c *DeploymentProcessConverterBase) generateStepOrderName(parent octopus.NameIdParentResource, owner octopus.NameIdParentResource) string {
	if parent != nil {
		return "process_step_order_" + sanitizer.SanitizeName(parent.GetName()) + "_" + sanitizer.SanitizeName(owner.GetName())
	}
	return "process_step_order_" + sanitizer.SanitizeName(owner.GetName())
}

func (c *DeploymentProcessConverterBase) getStepId(deploymentProcess octopus.OctopusProcess, runbookOrProject octopus.NameIdParentResource, step *octopus.Step) string {
	return runbookOrProject.GetId() + "/" + deploymentProcess.GetId() + "/" + strutil.EmptyIfNil(step.Id)
}

func (c *DeploymentProcessConverterBase) getActionId(deploymentProcess octopus.OctopusProcess, runbookOrProject octopus.NameIdParentResource, action *octopus.Action) string {
	return runbookOrProject.GetId() + "/" + deploymentProcess.GetId() + "/" + action.Id
}

func (c *DeploymentProcessConverterBase) generateChildStepOrderName(parent octopus.NameIdParentResource, owner octopus.NameIdParentResource, named octopus.NamedResource) string {
	if parent != nil {
		return "process_child_step_order_" + sanitizer.SanitizeName(parent.GetName()) + "_" + sanitizer.SanitizeName(owner.GetName()) + "_" + sanitizer.SanitizeName(named.GetName())
	}
	return "process_child_step_order_" + sanitizer.SanitizeName(owner.GetName()) + "_" + sanitizer.SanitizeName(named.GetName())
}

func (c *DeploymentProcessConverterBase) generateStepName(parent octopus.NameIdParentResource, owner octopus.NameIdParentResource, named octopus.NamedResource) string {
	if parent != nil {
		return "process_step_" + sanitizer.SanitizeName(parent.GetName()) + "_" + sanitizer.SanitizeName(owner.GetName()) + "_" + sanitizer.SanitizeName(named.GetName())
	}
	return "process_step_" + sanitizer.SanitizeName(owner.GetName()) + "_" + sanitizer.SanitizeName(named.GetName())
}

func (c *DeploymentProcessConverterBase) generateChildStepName(parent octopus.NameIdParentResource, owner octopus.NameIdParentResource, named octopus.NamedResource) string {
	if parent != nil {
		return "process_child_step_" + sanitizer.SanitizeName(parent.GetName()) + "_" + sanitizer.SanitizeName(owner.GetName()) + "_" + sanitizer.SanitizeName(named.GetName())
	}

	return "process_child_step_" + sanitizer.SanitizeName(owner.GetName()) + "_" + sanitizer.SanitizeName(named.GetName())
}

func (c *DeploymentProcessConverterBase) assignProperties(propertyName string, block *hclwrite.Block, owner octopus.NameIdParentResource, properties map[string]any, stepTemplateProperties []string, action octopus.NamedResource, file *hclwrite.File, dependencies *data.ResourceDetailsCollection) {
	if action == nil {
		return
	}

	sanitizedProperties, variables := steps.MapSanitizer{
		DummySecretGenerator:      c.DummySecretGenerator,
		DummySecretVariableValues: c.DummySecretVariableValues,
	}.SanitizeMap(owner, action, properties, dependencies)
	sanitizedProperties = c.OctopusActionProcessor.EscapeDollars(sanitizedProperties)
	sanitizedProperties = c.OctopusActionProcessor.EscapePercents(sanitizedProperties)
	sanitizedProperties = c.OctopusActionProcessor.ReplaceStepTemplateVersion(dependencies, sanitizedProperties)
	sanitizedProperties = c.OctopusActionProcessor.ReplaceIds(c.ExperimentalEnableStepTemplates, sanitizedProperties, dependencies)
	sanitizedProperties = c.OctopusActionProcessor.RemoveUnnecessaryActionFields(sanitizedProperties)
	sanitizedProperties = c.OctopusActionProcessor.RemoveFields(sanitizedProperties, stepTemplateProperties)
	sanitizedProperties = c.OctopusActionProcessor.RemoveStepTemplateFields(sanitizedProperties)
	sanitizedProperties = c.OctopusActionProcessor.FixActionFields(sanitizedProperties)
	sanitizedProperties = c.OctopusActionProcessor.LimitPropertyLength(c.LimitAttributeLength, true, sanitizedProperties)

	hcl.WriteStepProperties(propertyName, block, sanitizedProperties)

	for _, propertyVariables := range variables {
		propertyVariablesBlock := gohcl.EncodeAsBlock(propertyVariables, "variable")
		hcl.WriteUnquotedAttribute(propertyVariablesBlock, "type", "string")
		file.Body().AppendBlock(propertyVariablesBlock)
	}
}

func (c *DeploymentProcessConverterBase) assignPrimaryPackage(projectName string, terraformProcessStep *terraform.TerraformProcessStep, action *octopus.Action, file *hclwrite.File, dependencies *data.ResourceDetailsCollection) {
	primaryPackage, packageIdVariable := c.getPrimaryPackage(projectName, action, dependencies)

	if primaryPackage != nil {
		terraformProcessStep.PrimaryPackage = primaryPackage
		c.writeVariableToFile(file, packageIdVariable)
	}
}

func (c *DeploymentProcessConverterBase) assignReferencePackage(projectName string, terraformProcessStep *terraform.TerraformProcessStep, action *octopus.Action, file *hclwrite.File, dependencies *data.ResourceDetailsCollection) {
	referencePackages, referencePackageIdVariables := c.getPackages(projectName, action, dependencies)
	terraformProcessStep.Packages = referencePackages

	for _, variable := range referencePackageIdVariables {
		c.writeVariableToFile(file, variable)
	}
}

func (c *DeploymentProcessConverterBase) assignWorkerPool(terraformProcessStep terraform.TerraformStepWithWorkerPool, action *octopus.Action, file *hclwrite.File, dependencies *data.ResourceDetailsCollection) error {
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

	terraformProcessStep.SetWorkerPoolId(workerPool)

	return nil
}

// getPrimaryPackage returns the details of the primary package and an optional variable used to reference the package ID.
func (c *DeploymentProcessConverterBase) getPrimaryPackage(projectName string, action *octopus.Action, dependencies *data.ResourceDetailsCollection) (*terraform.TerraformProcessStepPackage, *terraform.TerraformVariable) {
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
func (c *DeploymentProcessConverterBase) getPackages(projectName string, action *octopus.Action, dependencies *data.ResourceDetailsCollection) (*map[string]terraform.TerraformProcessStepPackage, []*terraform.TerraformVariable) {
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

func (c *DeploymentProcessConverterBase) GetResourceType() string {
	return c.ResourceType
}

func (c *DeploymentProcessConverterBase) exportDependencies(recursive bool, lookup bool, stateless bool, resource octopus.OctopusProcess, dependencies *data.ResourceDetailsCollection) error {
	// Export linked accounts
	err := c.OctopusActionProcessor.ExportAccounts(recursive, lookup, stateless, resource.GetSteps(), dependencies)
	if err != nil {
		return err
	}

	// Export linked feeds
	err = c.OctopusActionProcessor.ExportFeeds(recursive, lookup, stateless, resource.GetSteps(), dependencies)
	if err != nil {
		return err
	}

	// Export linked worker pools
	err = c.OctopusActionProcessor.ExportWorkerPools(recursive, lookup, stateless, resource.GetSteps(), dependencies)
	if err != nil {
		return err
	}

	// Export linked environments
	err = c.OctopusActionProcessor.ExportEnvironments(recursive, lookup, stateless, resource.GetSteps(), dependencies)
	if err != nil {
		return err
	}

	// Export step templates
	err = c.OctopusActionProcessor.ExportStepTemplates(recursive, lookup, stateless, resource.GetSteps(), dependencies)
	if err != nil {
		return err
	}

	// Export git credentials
	err = c.OctopusActionProcessor.ExportGitCredentials(recursive, lookup, stateless, resource.GetSteps(), dependencies)
	if err != nil {
		return err
	}

	// Export projects, typically referenced in a "Deploy a release" step
	err = c.OctopusActionProcessor.ExportProjects(recursive, lookup, stateless, resource.GetSteps(), dependencies)
	if err != nil {
		return err
	}

	return nil
}

func (c *DeploymentProcessConverterBase) getPackageIdVariable(defaultValue string, projectName string, stepName string, packageName string) *terraform.TerraformVariable {
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

func (c *DeploymentProcessConverterBase) writeVariableToFile(file *hclwrite.File, variable *terraform.TerraformVariable) {
	if variable == nil {
		return
	}

	block := gohcl.EncodeAsBlock(variable, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)
}
