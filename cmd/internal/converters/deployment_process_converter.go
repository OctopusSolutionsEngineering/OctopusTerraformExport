package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/strutil"
)

type DeploymentProcessConverter struct {
	Client                 client.OctopusClient
	OctopusActionProcessor OctopusActionProcessor
	IgnoreProjectChanges   bool
}

func (c DeploymentProcessConverter) ToHclByIdAndName(id string, projectName string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.DeploymentProcess{}
	found, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	project := octopus.Project{}
	_, err = c.Client.GetResourceById("Projects", resource.ProjectId, &project)

	if err != nil {
		return err
	}

	return c.toHcl(resource, project.HasCacConfigured(), true, false, projectName, dependencies)
}

func (c DeploymentProcessConverter) ToHclLookupByIdAndName(id string, projectName string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.DeploymentProcess{}
	found, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Projects with no deployment process will not have a deployment process resources.
	// This is expected, so just return.
	if !found {
		return nil
	}

	project := octopus.Project{}
	_, err = c.Client.GetResourceById("Projects", resource.ProjectId, &project)

	if err != nil {
		return err
	}

	return c.toHcl(resource, project.HasCacConfigured(), false, true, projectName, dependencies)
}

func (c DeploymentProcessConverter) toHcl(resource octopus.DeploymentProcess, cac bool, recursive bool, lookup bool, projectName string, dependencies *ResourceDetailsCollection) error {
	resourceName := "deployment_process_" + sanitizer.SanitizeName(projectName)

	thisResource := ResourceDetails{}

	err := c.exportDependencies(recursive, lookup, resource, dependencies)

	if err != nil {
		return err
	}

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_deployment_process." + resourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformDeploymentProcess{
			Type:      "octopusdeploy_deployment_process",
			Name:      resourceName,
			ProjectId: dependencies.GetResource("Projects", resource.ProjectId),
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
				actionResource.Lookup = "${octopusdeploy_deployment_process." + resourceName + ".step[" + fmt.Sprint(i) + "].action[" + fmt.Sprint(j) + "].id}"
				dependencies.AddResource(actionResource)

				terraformResource.Step[i].Action[j] = terraform.TerraformAction{
					Name:                          a.Name,
					ActionType:                    a.ActionType,
					Notes:                         a.Notes,
					IsDisabled:                    a.IsDisabled,
					CanBeUsedForProjectVersioning: a.CanBeUsedForProjectVersioning,
					IsRequired:                    a.IsRequired,
					WorkerPoolId:                  dependencies.GetResource("WorkerPools", a.WorkerPoolId),
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

		if c.IgnoreProjectChanges || cac {
			all := "all"
			terraformResource.Lifecycle = &terraform.TerraformLifecycleMetaArgument{
				IgnoreAllChanges: &all,
			}
		}

		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		for _, s := range resource.Steps {
			for _, a := range s.Actions {
				properties := a.Properties
				sanitizedProperties := sanitizer.SanitizeMap(properties)
				sanitizedProperties = c.OctopusActionProcessor.EscapeDollars(sanitizedProperties)
				sanitizedProperties = c.OctopusActionProcessor.EscapePercents(sanitizedProperties)
				sanitizedProperties = c.OctopusActionProcessor.ReplaceIds(sanitizedProperties, dependencies)
				sanitizedProperties = c.OctopusActionProcessor.RemoveUnnecessaryActionFields(sanitizedProperties)
				sanitizedProperties = c.OctopusActionProcessor.DetachStepTemplates(sanitizedProperties)
				hcl.WriteActionProperties(block, *s.Name, *a.Name, sanitizedProperties)
			}
		}

		file.Body().AppendBlock(block)
		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c DeploymentProcessConverter) GetResourceType() string {
	return "DeploymentProcesses"
}

func (c DeploymentProcessConverter) exportDependencies(recursive bool, lookup bool, resource octopus.DeploymentProcess, dependencies *ResourceDetailsCollection) error {
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
