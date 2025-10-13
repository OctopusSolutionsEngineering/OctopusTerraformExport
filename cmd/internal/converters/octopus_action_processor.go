package converters

import (
	"fmt"
	"slices"
	"strings"

	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/regexes"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sliceutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/samber/lo"
)

// OctopusActionProcessor exposes a bunch of common functions for exporting the processes associated with
// projects and runbooks.
type OctopusActionProcessor struct {
	FeedConverter           ConverterAndLookupWithStatelessById
	AccountConverter        ConverterAndLookupWithStatelessById
	WorkerPoolConverter     ConverterAndLookupWithStatelessById
	EnvironmentConverter    ConverterAndLookupWithStatelessById
	GitCredentialsConverter ConverterAndLookupWithStatelessById
	ProjectExporter         ConverterAndLookupWithStatelessById
	DetachProjectTemplates  bool
	WorkerPoolProcessor     OctopusWorkerPoolProcessor
	StepTemplateConverter   ConverterAndLookupById
	Client                  client.OctopusClient
}

func (c OctopusActionProcessor) ExportFeeds(recursive bool, lookup bool, stateless bool, steps []octopus.Step, dependencies *data.ResourceDetailsCollection) error {

	if stateless {
		// Also export the built-in feed. This is useful for LLM training as it is expected to always exist.
		builtInFeed := octopus.Feed{}
		if found, err := c.Client.GetResourceByName("Feeds", "Octopus Server (built-in)", &builtInFeed); err != nil {
			return err
		} else if found {
			if err := c.FeedConverter.ToHclStatelessById(builtInFeed.Id, dependencies); err != nil {
				return err
			}
		}
	}

	for _, step := range steps {
		for _, action := range step.Actions {

			if strutil.NilIfEmptyPointer(action.Container.FeedId) != nil {
				if recursive {
					if stateless {
						if err := c.FeedConverter.ToHclStatelessById(strutil.EmptyIfNil(action.Container.FeedId), dependencies); err != nil {
							return err
						}
					} else {
						if err := c.FeedConverter.ToHclById(strutil.EmptyIfNil(action.Container.FeedId), dependencies); err != nil {
							return err
						}
					}
				} else if lookup {
					if err := c.FeedConverter.ToHclLookupById(strutil.EmptyIfNil(action.Container.FeedId), dependencies); err != nil {
						return err
					}
				}
			}

			for _, pack := range action.Packages {
				// We can have feed IDs that are octostache expressions. We don't process these further.
				if pack.FeedId != nil && regexes.FeedRegex.MatchString(strutil.EmptyIfNil(pack.FeedId)) {
					var err error
					if recursive {
						if stateless {
							err = c.FeedConverter.ToHclStatelessById(strutil.EmptyIfNil(pack.FeedId), dependencies)
						} else {
							err = c.FeedConverter.ToHclById(strutil.EmptyIfNil(pack.FeedId), dependencies)
						}
					} else if lookup {
						err = c.FeedConverter.ToHclLookupById(strutil.EmptyIfNil(pack.FeedId), dependencies)
					}

					if err != nil {
						return err
					}
				}
			}

			for _, prop := range action.Properties {
				for _, feed := range regexes.FeedRegex.FindAllString(fmt.Sprint(prop), -1) {
					var err error
					if recursive {
						if stateless {
							err = c.FeedConverter.ToHclStatelessById(feed, dependencies)
						} else {
							err = c.FeedConverter.ToHclById(feed, dependencies)
						}
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

func (c OctopusActionProcessor) ExportAccounts(recursive bool, lookup bool, stateless bool, steps []octopus.Step, dependencies *data.ResourceDetailsCollection) error {

	for _, step := range steps {
		for _, action := range step.Actions {
			for _, prop := range action.Properties {
				for _, account := range regexes.AccountRegex.FindAllString(fmt.Sprint(prop), -1) {
					var err error
					if recursive {
						if stateless {
							err = c.AccountConverter.ToHclStatelessById(account, dependencies)
						} else {
							err = c.AccountConverter.ToHclById(account, dependencies)
						}
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

func (c OctopusActionProcessor) ExportWorkerPools(recursive bool, lookup bool, stateless bool, steps []octopus.Step, dependencies *data.ResourceDetailsCollection) error {
	for _, step := range steps {
		for _, action := range step.Actions {
			workerPoolId, err := c.WorkerPoolProcessor.ResolveWorkerPoolId(action.WorkerPoolId)

			if err != nil {
				return err
			}

			if workerPoolId != "" {

				if recursive {
					if stateless {
						err = c.WorkerPoolConverter.ToHclStatelessById(workerPoolId, dependencies)
					} else {
						err = c.WorkerPoolConverter.ToHclById(workerPoolId, dependencies)
					}
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

func (c OctopusActionProcessor) ConvertContainer(container octopus.Container, dependencies *data.ResourceDetailsCollection) *terraform.TerraformProcessStepContainer {
	if container.Image != nil || container.FeedId != nil {
		return &terraform.TerraformProcessStepContainer{
			FeedId: dependencies.GetResourcePointer("Feeds", container.FeedId),
			Image:  container.Image,
		}
	}

	return nil
}

func (c OctopusActionProcessor) ConvertGitDependencies(gitDependencies []octopus.GitDependency, dependencies *data.ResourceDetailsCollection) []terraform.TerraformGitDependency {
	result := make([]terraform.TerraformGitDependency, len(gitDependencies))
	for i, gitDependency := range gitDependencies {
		result[i] = terraform.TerraformGitDependency{
			RepositoryUri:     gitDependency.RepositoryUri,
			DefaultBranch:     gitDependency.DefaultBranch,
			GitCredentialType: gitDependency.GitCredentialType,
			GitCredentialID:   dependencies.GetResourcePointer("Git-Credentials", gitDependency.GitCredentialId),
		}
	}
	return result
}

func (c OctopusActionProcessor) ConvertGitDependenciesV2(gitDependencies []octopus.GitDependency, dependencies *data.ResourceDetailsCollection) *map[string]terraform.TerraformProcessStepGitDependencies {
	result := map[string]terraform.TerraformProcessStepGitDependencies{}
	for _, gitDependency := range gitDependencies {
		result[strutil.EmptyIfNil(gitDependency.Name)] = terraform.TerraformProcessStepGitDependencies{
			DefaultBranch:     strutil.EmptyIfNil(gitDependency.DefaultBranch),
			GitCredentialType: strutil.EmptyIfNil(gitDependency.GitCredentialType),
			RepositoryUri:     strutil.EmptyIfNil(gitDependency.RepositoryUri),
			FilePathFilters:   nil,
			GitCredentialId:   dependencies.GetResourcePointer("Git-Credentials", gitDependency.GitCredentialId),
		}
	}

	if len(result) == 0 {
		return nil
	}

	return &result
}

func (c OctopusActionProcessor) ReplaceIds(properties map[string]string, dependencies *data.ResourceDetailsCollection) map[string]string {
	properties = c.replaceAccountIds(properties, dependencies)
	properties = c.replaceFeedIds(properties, dependencies)
	properties = c.replaceProjectIds(properties, dependencies)
	properties = c.replaceGitCredentialIds(properties, dependencies)
	properties = c.replaceStepTemplates(properties, dependencies)
	return properties
}

func (c OctopusActionProcessor) FixOctopusUseBundledTooling(actionType string, properties map[string]any) map[string]any {
	serverSteps := []string{
		"Octopus.Kubernetes.Kustomize",
		"Octopus.KubernetesDeployRawYaml",
	}

	sanitisedProperties := map[string]any{}
	for k, v := range properties {
		sanitisedProperties[k] = v
	}

	/*
		I've seen cases where manual interventions have Octopus.UseBundledTooling set to False
		when the step is created.

		Error: Provider produced inconsistent result after apply
		When applying changes to
		octopusdeploy_process_step.process_step_analytics_engine_5624_deploy_with_kustomize[0],
		provider "provider[\"registry.opentofu.org/octopusdeploy/octopusdeploy\"]"
		produced an unexpected new value: .execution_properties: new element
		"OctopusUseBundledTooling" has appeared.
	*/
	if slices.Contains(serverSteps, actionType) {
		if _, ok := sanitisedProperties["Octopus.UseBundledTooling"].(string); !ok {
			sanitisedProperties["Octopus.UseBundledTooling"] = "False"
		}
	}

	return sanitisedProperties
}

func (c OctopusActionProcessor) FixRunOnServer(actionType string, properties map[string]any) map[string]any {
	serverSteps := []string{
		"Octopus.Manual",
		"Octopus.Email",
		"Octopus.DeployRelease",
		"Octopus.JiraIntegration.ServiceDeskAction",
		"Octopus.HealthCheck",
		"Octopus.AzureResourceGroup",
		"Octopus.AwsRunCloudFormation",
		"Octopus.AwsDeleteCloudFormation",
		"Octopus.AwsApplyCloudFormationChangeSet",
		"Octopus.Kubernetes.Kustomize",
		"Octopus.HelmChartUpgrade",
	}

	sanitisedProperties := map[string]any{}
	for k, v := range properties {
		sanitisedProperties[k] = v
	}

	/*
		I've seen cases where manual interventions have Octopus.Action.RunOnServer set to false
		When this step is recreated, the value is true. This results in the error:

		Error: Provider produced inconsistent result after apply
		When applying changes to
		octopusdeploy_process_step.process_step_k8s_app_approve_production_deployment,
		provider "provider[\"registry.terraform.io/octopusdeploy/octopusdeploy\"]"
		produced an unexpected new value:
		.execution_properties["Octopus.Action.RunOnServer"]: was
		cty.StringVal("false"), but now cty.StringVal("true").

		And this one

		Error: Provider produced inconsistent result after apply
		When applying changes to
		octopusdeploy_process_step.process_step_azure_web_app_jira_service_desk_change_request,
		provider "provider[\"registry.terraform.io/octopusdeploy/octopusdeploy\"]"
		produced an unexpected new value: .execution_properties: inconsistent values
		for sensitive attribute.
		This is a bug in the provider, which should be reported in the provider's own
		issue tracker.
	*/
	if slices.Contains(serverSteps, actionType) {
		sanitisedProperties["Octopus.Action.RunOnServer"] = "true"
	}

	return sanitisedProperties
}

// EscapeDollars escapes variable interpolation
// https://developer.hashicorp.com/terraform/language/expressions/strings#escape-sequences
func (c OctopusActionProcessor) EscapeDollars(properties map[string]string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		sanitisedProperties[k] = strings.ReplaceAll(v, "${", "$${")
	}
	return sanitisedProperties
}

// EscapePercents escapes variable interpolation
// https://developer.hashicorp.com/terraform/language/expressions/strings#escape-sequences
func (c OctopusActionProcessor) EscapePercents(properties map[string]string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		sanitisedProperties[k] = strings.ReplaceAll(v, "%{", "%%{")
	}
	return sanitisedProperties
}

// FixActionFields deals with the case where the server returns lower case values for boolean properties
func (c OctopusActionProcessor) FixActionFields(properties map[string]string) map[string]string {
	return lo.MapValues(properties, func(value string, key string) string {
		/*
			Fix this error:
			When applying changes to
			octopusdeploy_process_step.process_step_test_runbook_hello_world__using_powershell_,
			provider "provider[\"registry.terraform.io/octopusdeploy/octopusdeploy\"]"
			produced an unexpected new value:
			.execution_properties["Octopus.Action.RunOnServer"]: was
			cty.StringVal("True"), but now cty.StringVal("true").
		*/

		if key == "Octopus.Action.RunOnServer" {
			return strings.ToLower(value)
		}
		return value
	})
}

// RemoveUnnecessaryActionFields removes generic property bag values that have more specific terraform properties
func (c OctopusActionProcessor) RemoveUnnecessaryActionFields(properties map[string]string) map[string]string {
	unnecessaryFields := []string{"Octopus.Action.Package.PackageId",
		// Fix up this error: .execution_properties: element "Octopus.Action.Package.DownloadOnTentacle" has vanished.
		"Octopus.Action.Package.DownloadOnTentacle",
		"Octopus.Action.Aws.CloudFormationTemplateParametersRaw",
		"Octopus.Action.Package.FeedId"}

	return c.RemoveFields(properties, unnecessaryFields)
}

func (c OctopusActionProcessor) RemoveStepTemplateFields(properties map[string]string) map[string]string {
	unnecessaryFields := []string{
		"Octopus.Action.Template.Id",
		"Octopus.Action.Template.Version",
	}
	return c.RemoveFields(properties, unnecessaryFields)
}

func (c OctopusActionProcessor) RemoveFields(properties map[string]string, unnecessaryFields []string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		if !sliceutil.Contains(unnecessaryFields, k) {
			sanitisedProperties[k] = v
		}
	}
	return sanitisedProperties
}

// LimitPropertyLength trims property bag values to a max length, if length is greater or equal to 0.
// If retainVariables is true, any variable references in the property are extracted and retained for context.
// The purpose of this is to reduce the length of the resulting HCL when used as context in a RAG query against
// an LLM. It has no valid use when generating HCL that is expected to be applied to an Octopus instance.
func (c OctopusActionProcessor) LimitPropertyLength(length int, retainVariables bool, properties map[string]string) map[string]string {
	if length <= 0 {
		return properties
	}

	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		sanitisedProperties[k] = LimitAttributeLength(length, retainVariables, v)
	}
	return sanitisedProperties
}

// ReplaceStepTemplateVersion replaces the step template version with a lookup value
func (c OctopusActionProcessor) ReplaceStepTemplateVersion(dependencies *data.ResourceDetailsCollection, properties map[string]string) map[string]string {
	if stepTemplate, ok := properties["Octopus.Action.Template.Id"]; ok {
		stepTemplateNewVersion := dependencies.GetResourceVersionLookup("ActionTemplates", stepTemplate)
		stepTemplateCurrentVersion := dependencies.GetResourceVersionCurrent("ActionTemplates", stepTemplate)

		if stepTemplateVersion, ok := properties["Octopus.Action.Template.Version"]; ok {
			if stepTemplateVersion == stepTemplateCurrentVersion {
				// If the version of the step template in the step is the same as the version of the step template
				// that is being exported, we know that this step references the latest step template. We should then
				// continue to reference the latest version after the step is recreated
				properties["Octopus.Action.Template.Version"] = stepTemplateNewVersion
			} else {
				// If the step referenced an older version of the step template, set the version to 0 to allow the newly
				// created step to show that an update is available. Technically we don't have a useful version to point
				// to here as step templates don't retain a history, and we just need a version that we know is not going
				// to be the current version of any newly imported step templates.
				// This does mean that newly created step templates need to be imported twice to ensure that the current
				// version is always at least 1, allowing us to indicate a previous version by setting this property to 0.
				properties["Octopus.Action.Template.Version"] = "0"
			}
		} else {
			properties["Octopus.Action.Template.Version"] = stepTemplateNewVersion
		}
	}

	return properties
}

// RemoveUnnecessaryStepFields removes generic property bag values that have more specific terraform properties
func (c OctopusActionProcessor) RemoveUnnecessaryStepFields(properties map[string]string) map[string]string {
	sanitisedProperties := map[string]string{}
	for k, v := range properties {
		if k != "Octopus.Action.TargetRoles" && k != "Octopus.Step.ConditionVariableExpression" {
			sanitisedProperties[k] = v
		}
	}
	return sanitisedProperties
}

// ReplaceFeedIds looks for any property value that is a valid feed ID and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c OctopusActionProcessor) replaceFeedIds(properties map[string]string, dependencies *data.ResourceDetailsCollection) map[string]string {
	for k, v := range properties {
		for _, v2 := range dependencies.GetAllResource("Feeds") {
			if len(v2.Id) != 0 && strings.Contains(v, v2.Id) {
				properties[k] = strings.ReplaceAll(v, v2.Id, v2.Lookup)
			}
		}
	}

	return properties
}

// replaceAccountIds looks for any property value that is a valid account ID and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c OctopusActionProcessor) replaceAccountIds(properties map[string]string, dependencies *data.ResourceDetailsCollection) map[string]string {
	for k, v := range properties {
		for _, v2 := range dependencies.GetAllResource("Accounts") {
			if len(v2.Id) != 0 && strings.Contains(v, v2.Id) {
				properties[k] = strings.ReplaceAll(v, v2.Id, v2.Lookup)
			}
		}
	}

	return properties
}

// replaceProjectIds looks for any property value that is a valid project ID and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c OctopusActionProcessor) replaceProjectIds(properties map[string]string, dependencies *data.ResourceDetailsCollection) map[string]string {
	for k, v := range properties {
		for _, v2 := range dependencies.GetAllResource("Projects") {
			if len(v2.Id) != 0 && strings.Contains(v, v2.Id) {
				properties[k] = strings.ReplaceAll(v, v2.Id, v2.Lookup)
			}
		}
	}

	return properties
}

// replaceGitCredentialIds looks for any property value that is a valid git credentials ID and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c OctopusActionProcessor) replaceGitCredentialIds(properties map[string]string, dependencies *data.ResourceDetailsCollection) map[string]string {
	for k, v := range properties {
		for _, v2 := range dependencies.GetAllResource("Git-Credentials") {
			if len(v2.Id) != 0 && strings.Contains(v, v2.Id) {
				properties[k] = strings.ReplaceAll(v, v2.Id, v2.Lookup)
			}
		}
	}

	return properties
}

// replaceStepTemplates looks for any property value that is a valid step template and replaces it with a resource ID lookup.
// This also looks in the property values, for instance when you export a JSON blob that has feed references.
func (c OctopusActionProcessor) replaceStepTemplates(properties map[string]string, dependencies *data.ResourceDetailsCollection) map[string]string {
	for k, v := range properties {
		for _, v2 := range dependencies.GetAllResource("ActionTemplates") {
			if len(v2.Id) != 0 && v == v2.Id {
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

func (c OctopusActionProcessor) ExportEnvironments(recursive bool, lookup bool, stateless bool, steps []octopus.Step, dependencies *data.ResourceDetailsCollection) error {
	for _, step := range steps {
		for _, action := range step.Actions {
			for _, environment := range lo.Union(action.Environments, action.ExcludedEnvironments) {
				var err error
				if recursive {
					if stateless {
						err = c.EnvironmentConverter.ToHclStatelessById(environment, dependencies)
					} else {
						err = c.EnvironmentConverter.ToHclById(environment, dependencies)
					}
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

func (c OctopusActionProcessor) ExportProjects(recursive bool, lookup bool, stateless bool, steps []octopus.Step, dependencies *data.ResourceDetailsCollection) error {
	for _, step := range steps {
		for _, action := range step.Actions {
			for _, prop := range action.Properties {
				for _, project := range regexes.ProjectsRegex.FindAllString(fmt.Sprint(prop), -1) {
					var err error
					if recursive {
						if stateless {
							err = c.ProjectExporter.ToHclStatelessById(project, dependencies)
						} else {
							err = c.ProjectExporter.ToHclById(project, dependencies)
						}
					} else if lookup {
						err = c.ProjectExporter.ToHclLookupById(project, dependencies)
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

func (c OctopusActionProcessor) ExportGitCredentials(recursive bool, lookup bool, stateless bool, steps []octopus.Step, dependencies *data.ResourceDetailsCollection) error {
	for _, step := range steps {
		for _, action := range step.Actions {
			for _, gitDependency := range action.GitDependencies {
				if gitDependency.GitCredentialId == nil {
					continue
				}

				var err error
				if recursive {
					if stateless {
						err = c.GitCredentialsConverter.ToHclStatelessById(strutil.EmptyIfNil(gitDependency.GitCredentialId), dependencies)
					} else {
						err = c.GitCredentialsConverter.ToHclById(strutil.EmptyIfNil(gitDependency.GitCredentialId), dependencies)
					}
				} else if lookup {
					err = c.GitCredentialsConverter.ToHclLookupById(strutil.EmptyIfNil(gitDependency.GitCredentialId), dependencies)
				}

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (c OctopusActionProcessor) ExportStepTemplates(recursive bool, lookup bool, stateless bool, steps []octopus.Step, dependencies *data.ResourceDetailsCollection) error {
	if c.DetachProjectTemplates {
		return nil
	}

	for _, step := range steps {
		for _, action := range step.Actions {
			for key, value := range action.Properties {
				if key == "Octopus.Action.Template.Id" {
					valueString := fmt.Sprint(value)

					var err error
					if recursive {
						if stateless {
							//err = c.StepTemplateConverter.ToHclStatelessById(valueString, dependencies)
						} else {
							err = c.StepTemplateConverter.ToHclById(valueString, dependencies)
						}
					} else if lookup {
						err = c.StepTemplateConverter.ToHclLookupById(valueString, dependencies)
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
