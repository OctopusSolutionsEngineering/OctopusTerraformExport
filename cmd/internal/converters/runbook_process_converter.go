package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	sanitizer2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sliceutil"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/strutil"
	"regexp"
	"strings"
)

type RunbookProcessConverter struct {
	Client              client.OctopusClient
	FeedConverter       ConverterAndLookupById
	AccountConverter    ConverterAndLookupById
	WorkerPoolConverter ConverterAndLookupById
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
				Properties:         c.removeUnnecessaryStepFields(c.replaceFeedIds(s.Properties, dependencies)),
				Condition:          s.Condition,
				StartTrigger:       s.StartTrigger,
				Action:             make([]terraform.TerraformAction, len(s.Actions)),
				TargetRoles:        c.getRoles(s.Properties),
			}

			for j, a := range s.Actions {

				actionResource := ResourceDetails{}
				actionResource.FileName = ""
				actionResource.Id = a.Id
				actionResource.ResourceType = "Actions"
				actionResource.Lookup = "${octopusdeploy_runbook_process." + resourceName + ".step[" + fmt.Sprint(i) + "].action[" + fmt.Sprint(j) + "].id}"
				dependencies.AddResource(actionResource)

				terraformResource.Step[i].Action[j] = terraform.TerraformAction{
					Name:                          a.Name,
					ActionType:                    a.ActionType,
					Notes:                         a.Notes,
					IsDisabled:                    a.IsDisabled,
					CanBeUsedForProjectVersioning: a.CanBeUsedForProjectVersioning,
					IsRequired:                    a.IsRequired,
					WorkerPoolId:                  dependencies.GetResource("WorkerPools", a.WorkerPoolId),
					Container:                     c.convertContainer(a.Container, dependencies),
					WorkerPoolVariable:            a.WorkerPoolVariable,
					Environments:                  dependencies.GetResources("Environments", a.Environments...),
					ExcludedEnvironments:          a.ExcludedEnvironments,
					Channels:                      a.Channels,
					TenantTags:                    a.TenantTags,
					Package:                       []terraform.TerraformPackage{},
					Condition:                     a.Condition,
					RunOnServer:                   c.getRunOnServer(a.Properties),
					Properties:                    nil,
					Features:                      c.getFeatures(a.Properties),
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
								Properties:              c.replaceIds(p.Properties, dependencies),
							})
					} else {
						terraformResource.Step[i].Action[j].PrimaryPackage = &terraform.TerraformPackage{
							Name:                    nil,
							PackageID:               p.PackageId,
							AcquisitionLocation:     p.AcquisitionLocation,
							ExtractDuringDeployment: nil,
							FeedId:                  dependencies.GetResourcePointer("Feeds", p.FeedId),
							Properties:              c.replaceIds(p.Properties, dependencies),
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
				sanitizedProperties = c.escapeDollars(sanitizedProperties)
				sanitizedProperties = c.escapePercents(sanitizedProperties)
				sanitizedProperties = c.replaceIds(sanitizedProperties, dependencies)
				sanitizedProperties = c.removeUnnecessaryActionFields(sanitizedProperties)
				hcl.WriteActionProperties(block, *s.Name, *a.Name, sanitizedProperties)
			}
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
	err := c.exportAccounts(recursive, lookup, resource, dependencies)
	if err != nil {
		return err
	}

	// Export linked feeds
	err = c.exportFeeds(recursive, lookup, resource, dependencies)
	if err != nil {
		return err
	}

	// Export linked worker pools
	err = c.exportWorkerPools(recursive, lookup, resource, dependencies)
	if err != nil {
		return err
	}

	return nil
}

func (c RunbookProcessConverter) exportFeeds(recursive bool, lookup bool, resource octopus.RunbookProcess, dependencies *ResourceDetailsCollection) error {
	feedRegex, _ := regexp.Compile("Feeds-\\d+")
	for _, step := range resource.Steps {
		for _, action := range step.Actions {

			if strutil.NilIfEmptyPointer(action.Container.FeedId) != nil {
				if recursive {
					c.FeedConverter.ToHclById(strutil.EmptyIfNil(action.Container.FeedId), dependencies)
				} else if lookup {
					c.FeedConverter.ToHclLookupById(strutil.EmptyIfNil(action.Container.FeedId), dependencies)
				}
			}

			for _, pack := range action.Packages {
				if pack.FeedId != nil {
					var err error
					if recursive {
						err = c.FeedConverter.ToHclById(strutil.EmptyIfNil(pack.FeedId), dependencies)
					} else if lookup {
						err = c.FeedConverter.ToHclLookupById(strutil.EmptyIfNil(pack.FeedId), dependencies)
					}

					if err != nil {
						return err
					}
				}
			}

			for _, prop := range action.Properties {
				for _, feed := range feedRegex.FindAllString(fmt.Sprint(prop), -1) {
					var err error
					if recursive {
						err = c.FeedConverter.ToHclById(feed, dependencies)
					} else if lookup {
						err = c.FeedConverter.ToHclLookupById(feed, dependencies)
					}

					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (c RunbookProcessConverter) exportAccounts(recursive bool, lookup bool, resource octopus.RunbookProcess, dependencies *ResourceDetailsCollection) error {
	accountRegex, _ := regexp.Compile("Accounts-\\d+")
	for _, step := range resource.Steps {
		for _, action := range step.Actions {
			for _, prop := range action.Properties {
				for _, account := range accountRegex.FindAllString(fmt.Sprint(prop), -1) {
					var err error
					if recursive {
						err = c.AccountConverter.ToHclById(account, dependencies)
					} else if lookup {
						err = c.AccountConverter.ToHclLookupById(account, dependencies)
					}

					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (c RunbookProcessConverter) exportWorkerPools(recursive bool, lookup bool, resource octopus.RunbookProcess, dependencies *ResourceDetailsCollection) error {
	for _, step := range resource.Steps {
		for _, action := range step.Actions {
			if action.WorkerPoolId != "" {
				var err error
				if recursive {
					err = c.WorkerPoolConverter.ToHclById(action.WorkerPoolId, dependencies)
				} else if lookup {
					err = c.WorkerPoolConverter.ToHclLookupById(action.WorkerPoolId, dependencies)
				}

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (c RunbookProcessConverter) convertContainer(container octopus.Container, dependencies *ResourceDetailsCollection) *terraform.TerraformContainer {
	if container.Image != nil || container.FeedId != nil {
		return &terraform.TerraformContainer{
			FeedId: dependencies.GetResourcePointer("Feeds", container.FeedId),
			Image:  container.Image,
		}
	}

	return nil
}

func (c RunbookProcessConverter) replaceIds(properties map[string]string, dependencies *ResourceDetailsCollection) map[string]string {
	return c.replaceFeedIds(c.replaceAccountIds(c.replaceAccountIds(properties, dependencies), dependencies), dependencies)
}

// https://developer.hashicorp.com/terraform/language/expressions/strings#escape-sequences
func (c RunbookProcessConverter) escapeDollars(properties map[string]string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		sanitisedProperties[k] = strings.ReplaceAll(v, "${", "$${")
	}
	return sanitisedProperties
}

// https://developer.hashicorp.com/terraform/language/expressions/strings#escape-sequences
func (c RunbookProcessConverter) escapePercents(properties map[string]string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		sanitisedProperties[k] = strings.ReplaceAll(v, "%{", "%%{")
	}
	return sanitisedProperties
}

// removeUnnecessaryActionFields removes generic property bag values that have more specific terraform properties
func (c RunbookProcessConverter) removeUnnecessaryActionFields(properties map[string]string) map[string]string {
	unnecessaryFields := []string{"Octopus.Action.Package.PackageId",
		"Octopus.Action.RunOnServer",
		"Octopus.Action.EnabledFeatures",
		"Octopus.Action.Aws.CloudFormationTemplateParametersRaw",
		"Octopus.Action.Package.FeedId"}
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		if !sliceutil.Contains(unnecessaryFields, k) {
			sanitisedProperties[k] = v
		}
	}
	return sanitisedProperties
}

// removeUnnecessaryActionFields removes generic property bag values that have more specific terraform properties
func (c RunbookProcessConverter) removeUnnecessaryStepFields(properties map[string]string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		if k != "Octopus.Action.TargetRoles" {
			sanitisedProperties[k] = v
		}
	}
	return sanitisedProperties
}

func (c RunbookProcessConverter) getRunOnServer(properties map[string]any) bool {
	v, ok := properties["Octopus.Action.RunOnServer"]
	if ok {
		return strings.ToLower(fmt.Sprint(v)) == "true"
	}

	return true
}

// replaceFeedIds looks for any property value that is a valid feed ID and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c RunbookProcessConverter) replaceFeedIds(properties map[string]string, dependencies *ResourceDetailsCollection) map[string]string {
	for k, v := range properties {
		for _, v2 := range dependencies.GetAllResource("Feeds") {
			if strings.Contains(v, v2.Id) {
				properties[k] = strings.ReplaceAll(v, v2.Id, v2.Lookup)
			}
		}
	}

	return properties
}

// replaceAccountIds looks for any property value that is a valid account ID and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c RunbookProcessConverter) replaceAccountIds(properties map[string]string, dependencies *ResourceDetailsCollection) map[string]string {
	for k, v := range properties {
		for _, v2 := range dependencies.GetAllResource("Accounts") {
			if strings.Contains(v, v2.Id) {
				properties[k] = strings.ReplaceAll(v, v2.Id, v2.Lookup)
			}
		}
	}

	return properties
}

func (c RunbookProcessConverter) getFeatures(properties map[string]any) []string {
	f, ok := properties["Octopus.Action.EnabledFeatures"]
	if ok {
		return strings.Split(fmt.Sprint(f), ",")
	}

	return []string{}
}

func (c RunbookProcessConverter) getRoles(properties map[string]string) []string {
	f, ok := properties["Octopus.Action.TargetRoles"]
	if ok {
		return strings.Split(fmt.Sprint(f), ",")
	}

	return []string{}
}