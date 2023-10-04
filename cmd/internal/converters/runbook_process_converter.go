package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	sanitizer2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

type RunbookProcessConverter struct {
	Client                 client.OctopusClient
	OctopusActionProcessor OctopusActionProcessor
	IgnoreProjectChanges   bool
	WorkerPoolProcessor    OctopusWorkerPoolProcessor
}

func (c RunbookProcessConverter) ToHclByIdAndName(id string, runbookName string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.RunbookProcess{}
	found, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	zap.L().Info("Runbook Process: " + resource.Id)
	return c.toHcl(resource, true, false, runbookName, dependencies)
}

func (c RunbookProcessConverter) ToHclLookupByIdAndName(id string, runbookName string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.RunbookProcess{}
	found, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	zap.L().Info("Runbook Process: " + resource.Id)
	return c.toHcl(resource, false, true, runbookName, dependencies)
}

func (c RunbookProcessConverter) toHcl(resource octopus.RunbookProcess, recursive bool, lookup bool, projectName string, dependencies *ResourceDetailsCollection) error {
	resourceName := "runbook_process_" + sanitizer2.SanitizeName(projectName)

	thisResource := ResourceDetails{}

	err := c.exportDependencies(recursive, lookup, resource, dependencies)

	if err != nil {
		return err
	}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_runbook_process." + resourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformRunbookProcess{
			Type:      "octopusdeploy_runbook_process",
			Name:      resourceName,
			RunbookId: dependencies.GetResource("Runbooks", resource.RunbookId),
			Step:      make([]terraform.TerraformStep, len(resource.Steps)),
		}

		for i, s := range resource.Steps {
			terraformResource.Step[i] = terraform.TerraformStep{
				Name:               s.Name,
				PackageRequirement: s.PackageRequirement,
				Properties:         c.OctopusActionProcessor.RemoveUnnecessaryStepFields(c.OctopusActionProcessor.ReplaceIds(s.Properties, dependencies)),
				Condition:          s.Condition,
				StartTrigger:       s.StartTrigger,
				Action:             make([]terraform.TerraformAction, len(s.Actions)),
				TargetRoles:        c.OctopusActionProcessor.GetRoles(s.Properties),
			}

			for j, a := range s.Actions {

				actionResource := ResourceDetails{}
				actionResource.FileName = ""
				actionResource.Id = a.Id
				actionResource.ResourceType = "Actions"
				actionResource.Lookup = "${octopusdeploy_runbook_process." + resourceName + ".step[" + fmt.Sprint(i) + "].action[" + fmt.Sprint(j) + "].id}"
				dependencies.AddResource(actionResource)

				workerPoolId, err := c.WorkerPoolProcessor.ResolveWorkerPoolId(a.WorkerPoolId)

				if err != nil {
					return "", err
				}

				terraformResource.Step[i].Action[j] = terraform.TerraformAction{
					Name:                          a.Name,
					ActionType:                    a.ActionType,
					Notes:                         a.Notes,
					IsDisabled:                    a.IsDisabled,
					CanBeUsedForProjectVersioning: a.CanBeUsedForProjectVersioning,
					IsRequired:                    a.IsRequired,
					WorkerPoolId:                  dependencies.GetResource("WorkerPools", workerPoolId),
					Container:                     c.OctopusActionProcessor.ConvertContainer(a.Container, dependencies),
					WorkerPoolVariable:            a.WorkerPoolVariable,
					Environments:                  dependencies.GetResources("Environments", a.Environments...),
					ExcludedEnvironments:          a.ExcludedEnvironments,
					Channels:                      a.Channels,
					TenantTags:                    a.TenantTags,
					Package:                       []terraform.TerraformPackage{},
					Condition:                     a.Condition,
					RunOnServer:                   c.OctopusActionProcessor.GetRunOnServer(a.Properties),
					Properties:                    nil,
					Features:                      c.OctopusActionProcessor.GetFeatures(a.Properties),
				}

				for _, p := range a.Packages {
					if strutil.NilIfEmptyPointer(p.Name) != nil {
						terraformResource.Step[i].Action[j].Package = append(
							terraformResource.Step[i].Action[j].Package,
							terraform.TerraformPackage{
								Name:                    p.Name,
								PackageID:               p.PackageId,
								AcquisitionLocation:     p.AcquisitionLocation,
								ExtractDuringDeployment: &p.ExtractDuringDeployment,
								FeedId:                  dependencies.GetResourcePointer("Feeds", p.FeedId),
								Properties:              c.OctopusActionProcessor.ReplaceIds(p.Properties, dependencies),
							})
					} else {
						terraformResource.Step[i].Action[j].PrimaryPackage = &terraform.TerraformPackage{
							Name:                    nil,
							PackageID:               p.PackageId,
							AcquisitionLocation:     p.AcquisitionLocation,
							ExtractDuringDeployment: nil,
							FeedId:                  dependencies.GetResourcePointer("Feeds", p.FeedId),
							Properties:              c.OctopusActionProcessor.ReplaceIds(p.Properties, dependencies),
						}
					}
				}
			}
		}

		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		for _, s := range resource.Steps {
			for _, a := range s.Actions {
				properties := a.Properties
				sanitizedProperties := sanitizer2.SanitizeMap(properties)
				sanitizedProperties = c.OctopusActionProcessor.EscapeDollars(sanitizedProperties)
				sanitizedProperties = c.OctopusActionProcessor.EscapePercents(sanitizedProperties)
				sanitizedProperties = c.OctopusActionProcessor.ReplaceIds(sanitizedProperties, dependencies)
				sanitizedProperties = c.OctopusActionProcessor.RemoveUnnecessaryActionFields(sanitizedProperties)
				sanitizedProperties = c.OctopusActionProcessor.DetachStepTemplates(sanitizedProperties)
				hcl.WriteActionProperties(block, *s.Name, *a.Name, sanitizedProperties)
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

func (c RunbookProcessConverter) exportDependencies(recursive bool, lookup bool, resource octopus.RunbookProcess, dependencies *ResourceDetailsCollection) error {
	// Export linked accounts
	err := c.OctopusActionProcessor.ExportAccounts(recursive, lookup, resource.Steps, dependencies)
	if err != nil {
		return err
	}

	// Export linked feeds
	err = c.OctopusActionProcessor.ExportFeeds(recursive, lookup, resource.Steps, dependencies)
	if err != nil {
		return err
	}

	// Export linked worker pools
	err = c.OctopusActionProcessor.ExportWorkerPools(recursive, lookup, resource.Steps, dependencies)
	if err != nil {
		return err
	}

	// Export linked environments
	err = c.OctopusActionProcessor.ExportEnvironments(recursive, lookup, resource.Steps, dependencies)
	if err != nil {
		return err
	}

	return nil
}
