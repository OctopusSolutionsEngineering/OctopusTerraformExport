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
	sanitizer2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type RunbookProcessConverter struct {
	Client                          client.OctopusClient
	OctopusActionProcessor          OctopusActionProcessor
	IgnoreProjectChanges            bool
	WorkerPoolProcessor             OctopusWorkerPoolProcessor
	ExcludeTenantTags               args.StringSliceArgs
	ExcludeTenantTagSets            args.StringSliceArgs
	Excluder                        ExcludeByName
	TagSetConverter                 ConvertToHclByResource[octopus.TagSet]
	LimitAttributeLength            int
	ExcludeAllSteps                 bool
	ExcludeSteps                    args.StringSliceArgs
	ExcludeStepsRegex               args.StringSliceArgs
	ExcludeStepsExcept              args.StringSliceArgs
	IgnoreInvalidExcludeExcept      bool
	ExperimentalEnableStepTemplates bool
	DummySecretGenerator            dummy.DummySecretGenerator
	DummySecretVariableValues       bool
}

func (c RunbookProcessConverter) ToHclByIdAndName(id string, runbookName string, recursive bool, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndName(id, runbookName, recursive, false, dependencies)
}

func (c RunbookProcessConverter) ToHclStatelessByIdAndName(id string, runbookName string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndName(id, runbookName, true, true, dependencies)
}

func (c RunbookProcessConverter) toHclByIdAndName(id string, runbookName string, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.RunbookProcess{}
	found, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.RunbookProcess: %w", err)
	}

	runbook := octopus.Runbook{}
	_, err = c.Client.GetSpaceResourceById("Runbooks", resource.RunbookId, &runbook)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Runbook: %w", err)
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	zap.L().Info("Runbook Process: " + resource.Id)
	return c.toHcl(resource, runbook.ProjectId, recursive, false, stateless, runbookName, dependencies)
}

func (c RunbookProcessConverter) ToHclLookupByIdAndName(id string, runbookName string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.RunbookProcess{}
	found, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.RunbookProcess: %w", err)
	}

	runbook := octopus.Runbook{}
	_, err = c.Client.GetSpaceResourceById("Runbooks", resource.RunbookId, &runbook)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.Runbook: %w", err)
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	zap.L().Info("Runbook Process: " + resource.Id)
	return c.toHcl(resource, runbook.ProjectId, false, true, false, runbookName, dependencies)
}

func (c RunbookProcessConverter) toHcl(resource octopus.RunbookProcess, projectId string, recursive bool, lookup bool, stateless bool, runbookName string, dependencies *data.ResourceDetailsCollection) error {
	resourceName := "runbook_process_" + sanitizer2.SanitizeName(runbookName)

	thisResource := data.ResourceDetails{}

	err := c.exportDependencies(recursive, lookup, stateless, resource, dependencies)

	if err != nil {
		return err
	}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_runbook_process." + resourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		validSteps := FilterSteps(
			resource.Steps,
			c.IgnoreInvalidExcludeExcept,
			c.Excluder,
			c.ExcludeAllSteps,
			c.ExcludeSteps,
			c.ExcludeStepsRegex,
			c.ExcludeStepsExcept)

		terraformResource := terraform.TerraformRunbookProcess{
			Type:      "octopusdeploy_runbook_process",
			Name:      resourceName,
			RunbookId: dependencies.GetResource("Runbooks", resource.RunbookId),
			Step:      make([]terraform.TerraformStep, len(validSteps)),
		}

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

				actionResource := data.ResourceDetails{}
				actionResource.FileName = ""
				actionResource.Id = a.Id
				actionResource.ResourceType = "Actions"
				actionResource.Lookup = "${octopusdeploy_runbook_process." + resourceName + ".step[" + fmt.Sprint(i) + "].action[" + fmt.Sprint(j) + "].id}"
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
					ExcludedEnvironments: a.ExcludedEnvironments,
					Channels:             a.Channels,
					TenantTags:           c.Excluder.FilteredTenantTags(a.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
					Package:              []terraform.TerraformPackage{},
					Condition:            a.Condition,
					RunOnServer:          c.OctopusActionProcessor.GetRunOnServer(a.Properties),
					Properties:           nil,
					Features:             c.OctopusActionProcessor.GetFeatures(a.Properties),
				}

				for _, p := range a.Packages {

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
								PackageID:               p.PackageId,
								AcquisitionLocation:     p.AcquisitionLocation,
								ExtractDuringDeployment: &p.ExtractDuringDeployment,
								FeedId:                  feedId,
								Properties:              c.OctopusActionProcessor.ReplaceIds(c.ExperimentalEnableStepTemplates, p.Properties, dependencies),
							})
					} else {
						terraformResource.Step[i].Action[j].PrimaryPackage = &terraform.TerraformPackage{
							Name:                    nil,
							PackageID:               p.PackageId,
							AcquisitionLocation:     p.AcquisitionLocation,
							ExtractDuringDeployment: nil,
							FeedId:                  feedId,
							Properties:              c.OctopusActionProcessor.ReplaceIds(c.ExperimentalEnableStepTemplates, p.Properties, dependencies),
						}
					}
				}
			}
		}

		if stateless {
			// only create the runbook process if the project was created
			terraformResource.Count = strutil.StrPointer(dependencies.GetResourceCount("Projects", projectId))
		}

		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "resource")
		allTenantTags := lo.FlatMap(resource.Steps, func(item octopus.Step, index int) []string {
			return lo.FlatMap(item.Actions, func(item octopus.Action, index int) []string {
				if item.TenantTags != nil {
					return item.TenantTags
				}
				return []string{}
			})
		})
		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, allTenantTags, c.TagSetConverter, block, dependencies, recursive)
		if err != nil {
			return "", err
		}

		for _, s := range validSteps {
			for _, a := range s.Actions {
				properties := a.Properties
				sanitizedProperties, variables := sanitizer2.MapSanitizer{
					DummySecretGenerator:      c.DummySecretGenerator,
					DummySecretVariableValues: c.DummySecretVariableValues,
				}.SanitizeMap(runbookName, strutil.EmptyIfNil(a.Name), properties, dependencies)
				sanitizedProperties = c.OctopusActionProcessor.EscapeDollars(sanitizedProperties)
				sanitizedProperties = c.OctopusActionProcessor.EscapePercents(sanitizedProperties)
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

		if c.IgnoreProjectChanges {
			hcl.WriteLifecycleAllAttribute(block)
		}

		file.Body().AppendBlock(block)
		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c RunbookProcessConverter) GetResourceType() string {
	return "RunbookProcesses"
}

func (c RunbookProcessConverter) exportDependencies(recursive bool, lookup bool, stateless bool, resource octopus.RunbookProcess, dependencies *data.ResourceDetailsCollection) error {
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

	return nil
}
