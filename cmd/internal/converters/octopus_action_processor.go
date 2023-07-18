package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sliceutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"regexp"
	"strings"
)

// OctopusActionProcessor exposes a bunch of common functions for exporting the processes associated with
// projects and runbooks.
type OctopusActionProcessor struct {
	FeedConverter          ConverterAndLookupById
	AccountConverter       ConverterAndLookupById
	WorkerPoolConverter    ConverterAndLookupById
	EnvironmentConverter   ConverterAndLookupById
	DetachProjectTemplates bool
	WorkerPoolProcessor    OctopusWorkerPoolProcessor
}

func (c OctopusActionProcessor) ExportFeeds(recursive bool, lookup bool, steps []octopus.Step, dependencies *ResourceDetailsCollection) error {
	feedRegex, _ := regexp.Compile("Feeds-\\d+")
	for _, step := range steps {
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

func (c OctopusActionProcessor) ExportAccounts(recursive bool, lookup bool, steps []octopus.Step, dependencies *ResourceDetailsCollection) error {
	accountRegex, _ := regexp.Compile("Accounts-\\d+")
	for _, step := range steps {
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

func (c OctopusActionProcessor) ExportWorkerPools(recursive bool, lookup bool, steps []octopus.Step, dependencies *ResourceDetailsCollection) error {
	for _, step := range steps {
		for _, action := range step.Actions {
			workerPoolId, err := c.WorkerPoolProcessor.ResolveWorkerPoolId(action.WorkerPoolId)

			if err != nil {
				return err
			}

			if workerPoolId != "" {

				if recursive {
					err = c.WorkerPoolConverter.ToHclById(workerPoolId, dependencies)
				} else if lookup {
					err = c.WorkerPoolConverter.ToHclLookupById(workerPoolId, dependencies)
				}

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (c OctopusActionProcessor) ConvertContainer(container octopus.Container, dependencies *ResourceDetailsCollection) *terraform.TerraformContainer {
	if container.Image != nil || container.FeedId != nil {
		return &terraform.TerraformContainer{
			FeedId: dependencies.GetResourcePointer("Feeds", container.FeedId),
			Image:  container.Image,
		}
	}

	return nil
}

func (c OctopusActionProcessor) ReplaceIds(properties map[string]string, dependencies *ResourceDetailsCollection) map[string]string {
	properties = c.replaceAccountIds(properties, dependencies)
	properties = c.replaceFeedIds(properties, dependencies)
	properties = c.replaceProjectIds(properties, dependencies)
	return properties
}

// https://developer.hashicorp.com/terraform/language/expressions/strings#escape-sequences
func (c OctopusActionProcessor) EscapeDollars(properties map[string]string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		sanitisedProperties[k] = strings.ReplaceAll(v, "${", "$${")
	}
	return sanitisedProperties
}

// https://developer.hashicorp.com/terraform/language/expressions/strings#escape-sequences
func (c OctopusActionProcessor) EscapePercents(properties map[string]string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		sanitisedProperties[k] = strings.ReplaceAll(v, "%{", "%%{")
	}
	return sanitisedProperties
}

// RemoveUnnecessaryActionFields removes generic property bag values that have more specific terraform properties
func (c OctopusActionProcessor) RemoveUnnecessaryActionFields(properties map[string]string) map[string]string {
	unnecessaryFields := []string{"Octopus.Action.Package.PackageId",
		// This value is usually redundant and specified by the run_on_server property, but it doesn't work for runbooks in 0.12.2
		// "Octopus.Action.RunOnServer",
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

// DetachStepTemplates detaches step templates, which is achieved by removing the template properties
func (c OctopusActionProcessor) DetachStepTemplates(properties map[string]string) map[string]string {
	if !c.DetachProjectTemplates {
		return properties
	}

	unnecessaryFields := []string{"Octopus.Action.Template.Id", "Octopus.Action.Template.Version"}
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		if !sliceutil.Contains(unnecessaryFields, k) {
			sanitisedProperties[k] = v
		}
	}
	return sanitisedProperties
}

// RemoveUnnecessaryStepFields removes generic property bag values that have more specific terraform properties
func (c OctopusActionProcessor) RemoveUnnecessaryStepFields(properties map[string]string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		if k != "Octopus.Action.TargetRoles" && v != "Octopus.Step.ConditionVariableExpression" {
			sanitisedProperties[k] = v
		}
	}
	return sanitisedProperties
}

func (c OctopusActionProcessor) GetRunOnServer(properties map[string]any) bool {
	v, ok := properties["Octopus.Action.RunOnServer"]
	if ok {
		return strings.ToLower(fmt.Sprint(v)) == "true"
	}

	return true
}

// ReplaceFeedIds looks for any property value that is a valid feed ID and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c OctopusActionProcessor) replaceFeedIds(properties map[string]string, dependencies *ResourceDetailsCollection) map[string]string {
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
func (c OctopusActionProcessor) replaceAccountIds(properties map[string]string, dependencies *ResourceDetailsCollection) map[string]string {
	for k, v := range properties {
		for _, v2 := range dependencies.GetAllResource("Accounts") {
			if strings.Contains(v, v2.Id) {
				properties[k] = strings.ReplaceAll(v, v2.Id, v2.Lookup)
			}
		}
	}

	return properties
}

// replaceProjectIds looks for any property value that is a valid project ID and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c OctopusActionProcessor) replaceProjectIds(properties map[string]string, dependencies *ResourceDetailsCollection) map[string]string {
	for k, v := range properties {
		for _, v2 := range dependencies.GetAllResource("Projects") {
			if strings.Contains(v, v2.Id) {
				properties[k] = strings.ReplaceAll(v, v2.Id, v2.Lookup)
			}
		}
	}

	return properties
}

func (c OctopusActionProcessor) GetFeatures(properties map[string]any) []string {
	f, ok := properties["Octopus.Action.EnabledFeatures"]
	if ok {
		return strings.Split(fmt.Sprint(f), ",")
	}

	return []string{}
}

func (c OctopusActionProcessor) GetRoles(properties map[string]string) []string {
	f, ok := properties["Octopus.Action.TargetRoles"]
	if ok {
		return strings.Split(fmt.Sprint(f), ",")
	}

	return []string{}
}

func (c OctopusActionProcessor) ExportEnvironments(recursive bool, lookup bool, steps []octopus.Step, dependencies *ResourceDetailsCollection) error {
	for _, step := range steps {
		for _, action := range step.Actions {
			for _, environment := range action.Environments {
				var err error
				if recursive {
					err = c.EnvironmentConverter.ToHclById(environment, dependencies)
				} else if lookup {
					err = c.EnvironmentConverter.ToHclLookupById(environment, dependencies)
				}

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
