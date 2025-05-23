package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/regexes"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/steps"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"net/url"
)

type DeploymentProcessConverter struct {
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

func (c *DeploymentProcessConverter) SetActionProcessor(actionProcessor *OctopusActionProcessor) {
	c.OctopusActionProcessor = actionProcessor
}

func (c *DeploymentProcessConverter) ToHclByIdAndBranch(parentId string, branch string, recursive bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndBranch(parentId, branch, recursive, false, dependencies)
}

func (c *DeploymentProcessConverter) ToHclStatelessByIdAndBranch(parentId string, branch string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndBranch(parentId, branch, true, true, dependencies)
}

func (c *DeploymentProcessConverter) toHclByIdAndBranch(parentId string, branch string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
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

func (c *DeploymentProcessConverter) ToHclLookupByIdAndBranch(parentId string, branch string, dependencies *data.ResourceDetailsCollection) error {
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

func (c *DeploymentProcessConverter) ToHclByIdAndName(id string, _ string, recursive bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndName(id, "", recursive, false, dependencies)
}

func (c *DeploymentProcessConverter) ToHclStatelessByIdAndName(id string, _ string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndName(id, "", true, true, dependencies)
}

func (c *DeploymentProcessConverter) toHclByIdAndName(id string, _ string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
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

func (c *DeploymentProcessConverter) ToHclLookupByIdAndName(id string, _ string, dependencies *data.ResourceDetailsCollection) error {
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

func (c *DeploymentProcessConverter) toHcl(resource octopus.DeploymentProcess, project octopus.Project, cac bool, recursive bool, lookup bool, stateless bool, projectName string, dependencies *data.ResourceDetailsCollection) error {
	resourceName := "deployment_process_" + sanitizer.SanitizeName(projectName)

	thisResource := data.ResourceDetails{}

	err := c.exportDependencies(recursive, lookup, stateless, resource, dependencies)

	if err != nil {
		return err
	}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_deployment_process." + resourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		validSteps := FilterSteps(
			resource.Steps,
			c.IgnoreInvalidExcludeExcept,
			c.Excluder,
			c.ExcludeAllSteps,
			c.ExcludeSteps,
			c.ExcludeStepsRegex,
			c.ExcludeStepsExcept)

		terraformResource := terraform.TerraformDeploymentProcess{
			Type:      "octopusdeploy_deployment_process",
			Name:      resourceName,
			ProjectId: dependencies.GetResource("Projects", resource.ProjectId),
			Step:      make([]terraform.TerraformStep, len(validSteps)),
		}

		file := hclwrite.NewEmptyFile()

		for i, s := range validSteps {
			terraformResource.Step[i] = terraform.TerraformStep{
				Name:                s.Name,
				PackageRequirement:  s.PackageRequirement,
				Properties:          c.OctopusActionProcessor.RemoveUnnecessaryStepFields(c.OctopusActionProcessor.ReplaceIds(c.ExperimentalEnableStepTemplates, s.Properties, dependencies)),
				Condition:           s.Condition,
				ConditionExpression: strutil.NilIfEmpty(s.Properties["Octopus.Step.ConditionVariableExpression"]),
				StartTrigger:        s.StartTrigger,
				Action:              make([]terraform.TerraformAction, len(s.Actions)),
				TargetRoles:         c.OctopusActionProcessor.GetRoles(s.Properties),
			}

			for j, a := range s.Actions {
				// Don't import duplicates
				if dependencies.HasResource(a.GenerateDeploymentProcessId(&resource), "Actions") {
					continue
				}

				actionResource := data.ResourceDetails{}
				actionResource.FileName = ""
				actionResource.Id = a.GenerateDeploymentProcessId(&resource)
				actionResource.ResourceType = "Actions"
				actionResource.Lookup = "${octopusdeploy_deployment_process." + resourceName + ".step[" + fmt.Sprint(i) + "].action[" + fmt.Sprint(j) + "].id}"
				dependencies.AddResource(actionResource)

				workerPoolId, err := c.WorkerPoolProcessor.ResolveWorkerPoolId(a.WorkerPoolId)

				if err != nil {
					return "", err
				}

				// don't lookup empty worker pool values
				workerPool := ""
				if len(workerPoolId) != 0 {
					workerPool = dependencies.GetResource("WorkerPools", workerPoolId)
				}

				if strutil.EmptyIfNil(s.Name) == "Email Team of Status (Always Run)" {
					zap.L().Warn("Action type is nil")
				}

				terraformResource.Step[i].Action[j] = terraform.TerraformAction{
					Name:                          a.Name,
					ActionType:                    a.ActionType,
					Notes:                         a.Notes,
					IsDisabled:                    a.IsDisabled,
					CanBeUsedForProjectVersioning: a.CanBeUsedForProjectVersioning,
					IsRequired:                    a.IsRequired,
					WorkerPoolId:                  strutil.InputIfEnabled(a.WorkerPoolVariable == nil, workerPool),
					Container:                     c.OctopusActionProcessor.ConvertContainer(a.Container, dependencies),
					// an empty string caused problems, so we need to return nil here for an empty string
					WorkerPoolVariable:   strutil.NilIfEmptyPointer(a.WorkerPoolVariable),
					Environments:         dependencies.GetResources("Environments", a.Environments...),
					ExcludedEnvironments: dependencies.GetResources("Environments", a.ExcludedEnvironments...),
					Channels:             dependencies.GetResources("Channels", a.Channels...),
					TenantTags:           c.Excluder.FilteredTenantTags(a.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
					Package:              []terraform.TerraformPackage{},
					Condition:            a.Condition,
					RunOnServer:          c.OctopusActionProcessor.GetRunOnServer(a.Properties),
					Properties:           nil,
					Features:             c.OctopusActionProcessor.GetFeatures(a.Properties),
					GitDependencies:      c.OctopusActionProcessor.ConvertGitDependencies(a.GitDependencies, dependencies),
				}

				for _, p := range a.Packages {
					packageId := strutil.EmptyIfNil(p.PackageId)

					var variableReference string

					// packages can be project IDs when they are defined in a "Deploy a release" step
					if regexes.ProjectsRegex.MatchString(packageId) && strutil.EmptyIfNil(a.ActionType) == "Octopus.DeployRelease" {
						variableReference = dependencies.GetResource("Projects", packageId)
					} else {
						variableReference = c.writePackageIdVariable(
							file,
							packageId,
							projectName,
							strutil.EmptyIfNil(a.Name),
							strutil.EmptyIfNil(p.Name))
					}

					// Don't look up a feed id that is a variable reference
					feedId := p.FeedId
					if regexes.FeedRegex.MatchString(strutil.EmptyIfNil(feedId)) {
						feedId = dependencies.GetResourcePointer("Feeds", p.FeedId)
					}

					if strutil.NilIfEmptyPointer(p.Name) != nil {
						terraformResource.Step[i].Action[j].Package = append(
							terraformResource.Step[i].Action[j].Package,
							terraform.TerraformPackage{
								Name:                    p.Name,
								PackageID:               &variableReference,
								AcquisitionLocation:     p.AcquisitionLocation,
								ExtractDuringDeployment: &p.ExtractDuringDeployment,
								FeedId:                  feedId,
								Properties:              c.OctopusActionProcessor.ReplaceIds(c.ExperimentalEnableStepTemplates, p.Properties, dependencies),
							})
					} else {
						terraformResource.Step[i].Action[j].PrimaryPackage = &terraform.TerraformPackage{
							Name:                    nil,
							PackageID:               &variableReference,
							AcquisitionLocation:     p.AcquisitionLocation,
							ExtractDuringDeployment: nil,
							FeedId:                  feedId,
							Properties:              c.OctopusActionProcessor.ReplaceIds(c.ExperimentalEnableStepTemplates, p.Properties, dependencies),
						}
					}
				}
			}
		}

		allTenantTags := lo.FlatMap(resource.Steps, func(item octopus.Step, index int) []string {
			return lo.FlatMap(item.Actions, func(item octopus.Action, index int) []string {
				if item.TenantTags != nil {
					return item.TenantTags
				}
				return []string{}
			})
		})

		if stateless {
			// only create the deployment process if the project was created
			terraformResource.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", project.Id))
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")
		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, allTenantTags, c.TagSetConverter, block, dependencies, recursive)
		if err != nil {
			return "", err
		}

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		for _, s := range validSteps {
			for _, a := range s.Actions {
				properties := a.Properties
				sanitizedProperties, variables := steps.MapSanitizer{
					DummySecretGenerator:      c.DummySecretGenerator,
					DummySecretVariableValues: c.DummySecretVariableValues,
				}.SanitizeMap(project, a, properties, dependencies)
				sanitizedProperties = c.OctopusActionProcessor.EscapeDollars(sanitizedProperties)
				sanitizedProperties = c.OctopusActionProcessor.EscapePercents(sanitizedProperties)
				sanitizedProperties = c.OctopusActionProcessor.ReplaceStepTemplateVersion(dependencies, sanitizedProperties)
				sanitizedProperties = c.OctopusActionProcessor.ReplaceIds(c.ExperimentalEnableStepTemplates, sanitizedProperties, dependencies)
				sanitizedProperties = c.OctopusActionProcessor.RemoveUnnecessaryActionFields(sanitizedProperties)
				sanitizedProperties = c.OctopusActionProcessor.DetachStepTemplates(sanitizedProperties)
				sanitizedProperties = c.OctopusActionProcessor.LimitPropertyLength(c.LimitAttributeLength, true, sanitizedProperties)

				hcl.WriteActionProperties(block, *s.Name, *a.Name, sanitizedProperties)

				for _, propertyVariables := range variables {
					propertyVariablesBlock := gohcl.EncodeAsBlock(propertyVariables, "variable")
					hcl.WriteUnquotedAttribute(propertyVariablesBlock, "type", "string")
					file.Body().AppendBlock(propertyVariablesBlock)
				}
			}
		}

		file.Body().AppendBlock(block)
		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c *DeploymentProcessConverter) GetResourceType() string {
	return "DeploymentProcesses"
}

func (c *DeploymentProcessConverter) exportDependencies(recursive bool, lookup bool, stateless bool, resource octopus.DeploymentProcess, dependencies *data.ResourceDetailsCollection) error {
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

func (c *DeploymentProcessConverter) writePackageIdVariable(file *hclwrite.File, defaultValue string, projectName string, stepName string, packageName string) string {
	if c.ExcludeTerraformVariables {
		return defaultValue
	}

	sanitizedProjectName := sanitizer.SanitizeName(projectName)
	sanitizedPackageName := sanitizer.SanitizeName(packageName)
	sanitizedStepName := sanitizer.SanitizeName(stepName)

	variableName := ""

	if packageName == "" {
		variableName = "project_" + sanitizedProjectName + "_step_" + sanitizedStepName + "_packageid"
	} else {
		variableName = "project_" + sanitizedProjectName + "_step_" + sanitizedStepName + "_package_" + sanitizedPackageName + "_packageid"
	}

	secretVariableResource := terraform.TerraformVariable{
		Name:        variableName,
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The package ID for the package named " + packageName + " from step " + stepName + " in project " + projectName,
		Default:     &defaultValue,
	}

	block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	return "${var." + variableName + "}"
}
