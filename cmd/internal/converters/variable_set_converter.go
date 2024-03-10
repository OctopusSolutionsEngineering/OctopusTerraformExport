package converters

import (
	"errors"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/regexes"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"k8s.io/utils/strings/slices"
	"net/url"
	"strings"
)

const octopusdeployVariableResourceType = "octopusdeploy_variable"

// VariableSetConverter exports variable sets.
// Note that we only access variable sets as dependencies of other resources, like project variables or
// library variable sets. There is no global collection or all endpoint that we can use to dump variables
// in bulk.
type VariableSetConverter struct {
	Client                              client.OctopusClient
	ChannelConverter                    ConverterByProjectIdWithTerraDependencies
	EnvironmentConverter                ConverterAndLookupWithStatelessById
	TagSetConverter                     ConvertToHclByResource[octopus.TagSet]
	AzureCloudServiceTargetConverter    ConverterAndLookupWithStatelessById
	AzureServiceFabricTargetConverter   ConverterAndLookupWithStatelessById
	AzureWebAppTargetConverter          ConverterAndLookupWithStatelessById
	CloudRegionTargetConverter          ConverterAndLookupWithStatelessById
	KubernetesTargetConverter           ConverterAndLookupWithStatelessById
	ListeningTargetConverter            ConverterAndLookupWithStatelessById
	OfflineDropTargetConverter          ConverterAndLookupWithStatelessById
	PollingTargetConverter              ConverterAndLookupWithStatelessById
	SshTargetConverter                  ConverterAndLookupWithStatelessById
	AccountConverter                    ConverterAndLookupWithStatelessById
	FeedConverter                       ConverterAndLookupWithStatelessById
	CertificateConverter                ConverterAndLookupWithStatelessById
	WorkerPoolConverter                 ConverterAndLookupWithStatelessById
	IgnoreCacManagedValues              bool
	DefaultSecretVariableValues         bool
	DummySecretVariableValues           bool
	ExcludeAllProjectVariables          bool
	ExcludeProjectVariables             args.ExcludeVariables
	ExcludeProjectVariablesExcept       args.ExcludeVariables
	ExcludeProjectVariablesRegex        args.ExcludeVariables
	IgnoreProjectChanges                bool
	DummySecretGenerator                DummySecretGenerator
	ExcludeVariableEnvironmentScopes    args.ExcludeVariableEnvironmentScopes
	excludeVariableEnvironmentScopesIds []string
	Excluder                            ExcludeByName
	ErrGroup                            *errgroup.Group
}

func (c *VariableSetConverter) ToHclByProjectIdBranchAndName(projectId string, branch string, parentName string, parentLookup string, parentCount *string, dependencies *data.ResourceDetailsCollection) error {
	if projectId == "" {
		return nil
	}

	resource := octopus.VariableSet{}
	found, err := c.Client.GetResource("Projects/"+projectId+"/"+url.QueryEscape(branch)+"/variables", &resource)

	if err != nil {
		return err
	}

	if !found {
		return nil
	}

	project := octopus.Project{}
	_, err = c.Client.GetResourceById("Projects", projectId, &project)

	if err != nil {
		return err
	}

	ignoreSecrets := project.HasCacConfigured() && c.IgnoreCacManagedValues

	zap.L().Info("VariableSet: " + strutil.EmptyIfNil(resource.Id))
	return c.toHcl(resource, false, true, false, ignoreSecrets, parentName, parentLookup, nil, dependencies)
}

func (c *VariableSetConverter) ToHclLookupByProjectIdBranchAndName(projectId string, branch string, parentName string, parentLookup string, dependencies *data.ResourceDetailsCollection) error {
	if projectId == "" {
		return nil
	}

	resource := octopus.VariableSet{}
	found, err := c.Client.GetResource("Projects/"+projectId+"/"+url.QueryEscape(branch)+"/variables", &resource)

	if err != nil {
		return err
	}

	if !found {
		return nil
	}

	project := octopus.Project{}
	_, err = c.Client.GetResourceById("Projects", projectId, &project)

	if err != nil {
		return err
	}

	ignoreSecrets := project.HasCacConfigured() && c.IgnoreCacManagedValues

	zap.L().Info("VariableSet: " + strutil.EmptyIfNil(resource.Id))
	return c.toHcl(resource, false, true, false, ignoreSecrets, parentName, parentLookup, nil, dependencies)
}

// ToHclByProjectIdAndName is called when returning variables from projects. This is because the variable set ID
// defined on a CaC enabled project is not available from the global /variablesets endpoint, and can only be
// accessed from the project resource.
func (c *VariableSetConverter) ToHclByProjectIdAndName(projectId string, parentName string, parentLookup string, parentCount *string, dependencies *data.ResourceDetailsCollection) error {
	if projectId == "" {
		return nil
	}

	resource := octopus.VariableSet{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &resource)

	if err != nil {
		return err
	}

	project := octopus.Project{}
	_, err = c.Client.GetResourceById("Projects", projectId, &project)

	if err != nil {
		return err
	}

	ignoreSecrets := project.HasCacConfigured() && c.IgnoreCacManagedValues

	zap.L().Info("VariableSet: " + strutil.EmptyIfNil(resource.Id))
	return c.toHcl(resource, false, false, parentCount != nil, ignoreSecrets, parentName, parentLookup, parentCount, dependencies)
}

func (c *VariableSetConverter) ToHclLookupByProjectIdAndName(projectId string, parentName string, parentLookup string, dependencies *data.ResourceDetailsCollection) error {
	if projectId == "" {
		return nil
	}

	resource := octopus.VariableSet{}
	_, err := c.Client.GetResource(c.GetGroupResourceType(projectId), &resource)

	if err != nil {
		return err
	}

	project := octopus.Project{}
	_, err = c.Client.GetResourceById("Projects", projectId, &project)

	if err != nil {
		return err
	}

	ignoreSecrets := project.HasCacConfigured() && c.IgnoreCacManagedValues

	zap.L().Info("VariableSet: " + strutil.EmptyIfNil(resource.Id))
	return c.toHcl(resource, false, true, false, ignoreSecrets, parentName, parentLookup, nil, dependencies)
}

func (c *VariableSetConverter) ToHclByIdAndName(id string, recursive bool, parentName string, parentLookup string, parentCount *string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndName(id, recursive, false, parentName, parentLookup, parentCount, dependencies)
}

func (c *VariableSetConverter) ToHclStatelessByIdAndName(id string, recursive bool, parentName string, parentLookup string, parentCount *string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclByIdAndName(id, recursive, true, parentName, parentLookup, parentCount, dependencies)
}

func (c *VariableSetConverter) toHclByIdAndName(id string, recursive bool, stateless bool, parentName string, parentLookup string, parentCount *string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.VariableSet{}
	found, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Some CaC enabled projects have no variable set.
	// This is expected, so just return.
	if !found {
		return nil
	}

	zap.L().Info("VariableSet: " + strutil.EmptyIfNil(resource.Id))
	return c.toHcl(resource, recursive, false, stateless, false, parentName, parentLookup, parentCount, dependencies)
}

// ToHclLookupByIdAndName exports the variable set as a complete resource, but will reference external resources like accounts,
// feeds, worker pools, certificates, environments, and targets as data source lookups.
func (c *VariableSetConverter) ToHclLookupByIdAndName(id string, parentName string, parentLookup string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.VariableSet{}
	found, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Some CaC enabled projects have no variable set.
	// This is expected, so just return.
	if !found {
		return nil
	}

	zap.L().Info("VariableSet: " + strutil.EmptyIfNil(resource.Id))
	return c.toHcl(resource, false, true, false, false, parentName, parentLookup, nil, dependencies)
}

func (c *VariableSetConverter) toHcl(resource octopus.VariableSet, recursive bool, lookup bool, stateless bool, ignoreSecrets bool, parentName string, parentLookup string, parentCount *string, dependencies *data.ResourceDetailsCollection) error {
	c.convertEnvironmentsToIds()

	nameCount := map[string]int{}
	for _, v := range resource.Variables {
		// Do not export regular variables if ignoring cac managed values
		if ignoreSecrets && !v.IsSensitive {
			continue
		}

		// Do not export excluded variables
		if c.Excluder.IsResourceExcludedWithRegex(v.Name, c.ExcludeAllProjectVariables, c.ExcludeProjectVariables, c.ExcludeProjectVariablesRegex, c.ExcludeProjectVariablesExcept) {
			continue
		}

		// Generate a unique suffix for each variable name
		if count, ok := nameCount[v.Name]; ok {
			nameCount[v.Name] = count + 1
		} else {
			nameCount[v.Name] = 1
		}

		v := v
		file := hclwrite.NewEmptyFile()
		thisResource := data.ResourceDetails{}

		resourceName := sanitizer.SanitizeName(parentName) + "_" + sanitizer.SanitizeName(v.Name) + "_" + fmt.Sprint(nameCount[v.Name])

		// Export linked accounts
		err := c.exportAccounts(recursive, lookup, stateless, v.Value, dependencies)
		if err != nil {
			return err
		}

		// Export linked feeds
		err = c.exportFeeds(recursive, lookup, stateless, v.Value, dependencies)
		if err != nil {
			return err
		}

		// Export linked worker pools
		err = c.exportWorkerPools(recursive, lookup, stateless, v.Value, dependencies)
		if err != nil {
			return err
		}

		// Export linked certificates
		err = c.exportCertificates(recursive, lookup, stateless, v.Value, dependencies)
		if err != nil {
			return err
		}

		// Export linked environments
		err = c.exportEnvironments(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export azure cloud service targets
		err = c.exportAzureCloudServiceTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export azure service fabric targets
		err = c.exportAzureServiceFabricTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export azure web app targets
		err = c.exportAzureWebAppTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export azure web app targets
		err = c.exportCloudRegionTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export kubernetes targets
		err = c.exportKubernetesTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export listening targets
		err = c.exportListeningTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export listening targets
		err = c.exportOfflineDropTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export polling targets
		err = c.exportPollingTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Export polling targets
		err = c.exportSshTargets(recursive, lookup, stateless, &v, dependencies)
		if err != nil {
			return err
		}

		// Placing sensitive variables in uniquely prefixed files allows us to target them for variable substitution
		if v.IsSensitive {
			thisResource.FileName = "space_population/project_variable_sensitive_" + resourceName + ".tf"
		} else {
			thisResource.FileName = "space_population/project_variable_" + resourceName + ".tf"
		}

		thisResource.Id = v.Id
		thisResource.Name = v.Name
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${" + octopusdeployVariableResourceType + "." + resourceName + ".id}"
		if v.IsSensitive {
			thisResource.Parameters = []data.ResourceParameter{
				{
					Label: "Sensitive variable " + v.Name + " password",
					Description: "The sensitive value associated with the variable \"" + v.Name + "\" in the belonging to " +
						parentName + v.Scope.ScopeDescription(" (", ")", dependencies),
					ResourceName:  sanitizer.SanitizeParameterName(dependencies, v.Name, "SensitiveValue"),
					Sensitive:     true,
					VariableName:  resourceName,
					ParameterType: "SensitiveValue",
				},
			}
		}
		thisResource.ToHcl = func() (string, error) {

			// Replace anything that looks like an octopus resource reference
			value := c.getAccount(v.Value, dependencies)
			value = c.getFeeds(value, dependencies)
			value = c.getCertificates(value, dependencies)
			value = c.getWorkerPools(value, dependencies)

			terraformResource := terraform.TerraformProjectVariable{
				Name:           resourceName,
				Type:           octopusdeployVariableResourceType,
				Count:          parentCount,
				OwnerId:        parentLookup,
				Value:          value,
				ResourceName:   v.Name,
				ResourceType:   v.Type,
				Description:    v.Description,
				SensitiveValue: c.convertSecretValue(v, resourceName),
				IsSensitive:    v.IsSensitive,
				Prompt:         c.convertPrompt(v.Prompt),
				Scope:          c.convertScope(v, dependencies),
			}

			if v.IsSensitive {
				var defaultValue *string = nil

				// Dummy values take precedence over default values
				if c.DummySecretVariableValues {
					defaultValue = c.DummySecretGenerator.GetDummySecret()
				} else if c.DefaultSecretVariableValues {
					defaultValueLookup := "#{" + v.Name + "}"
					defaultValue = &defaultValueLookup
				}

				secretVariableResource := terraform.TerraformVariable{
					Name:        resourceName,
					Type:        "string",
					Nullable:    true,
					Sensitive:   true,
					Description: "The secret variable value associated with the variable " + v.Name,
					Default:     defaultValue,
				}

				block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
				hcl.WriteUnquotedAttribute(block, "type", "string")

				file.Body().AppendBlock(block)
			} else if v.Type == "String" && !hcl.IsInterpolation(strutil.EmptyIfNil(value)) {
				// Use a second terraform variable to allow the octopus variable to be defined at apply time.
				// Note this only applies to string variables, as other types likely reference resources
				// that are being created by terraform, and these dynamic values can not be used as default
				// variable values.
				terraformResource.Value = c.convertValue(v, resourceName)
				regularVariable := terraform.TerraformVariable{
					Name:        resourceName,
					Type:        "string",
					Nullable:    true,
					Sensitive:   false,
					Description: "The value associated with the variable " + v.Name,
					Default:     strutil.StrPointer(strutil.EmptyIfNil(value)),
				}

				block := gohcl.EncodeAsBlock(regularVariable, "variable")
				hcl.WriteUnquotedAttribute(block, "type", "string")
				file.Body().AppendBlock(block)
			}

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			// When using dummy values, we expect the secrets will be updated later
			if c.DummySecretVariableValues || stateless {

				ignoreAll := terraform.EmptyBlock{}
				lifecycleBlock := gohcl.EncodeAsBlock(ignoreAll, "lifecycle")
				block.Body().AppendBlock(lifecycleBlock)

				if c.DummySecretVariableValues {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "ignore_changes", "[sensitive_value]")
				}

				if stateless {
					hcl.WriteUnquotedAttribute(lifecycleBlock, "prevent_destroy", "true")
				}
			}

			// If we are creating the tag sets (i.e. exporting a space or recursively exporting a project),
			// ensure tag sets are create before the variable.
			// If we are doing a lookup, the tag sets are expected to already be available, and so there is
			// no dependency relationship.
			if !lookup {
				tagSetDependencies, err := c.addTagSetDependencies(v, recursive, dependencies)

				if err != nil {
					return "", err
				}

				// Explicitly describe the dependency between a variable and a tag set
				dependsOn := []string{}
				for resourceType, terraformDependencies := range tagSetDependencies {
					for _, terraformDependency := range terraformDependencies {
						dependency := dependencies.GetResourceDependency(resourceType, terraformDependency)
						dependency = hcl.RemoveId(hcl.RemoveInterpolation(dependency))
						dependsOn = append(dependsOn, dependency)
					}
				}
				hcl.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(dependsOn[:], ",")+"]")
			}

			// Ignore all changes if requested
			if c.IgnoreProjectChanges {
				hcl.WriteLifecycleAllAttribute(block)
			}

			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c *VariableSetConverter) GetResourceType() string {
	return "Variables"
}

func (c *VariableSetConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/Variables"
}

func (c *VariableSetConverter) convertSecretValue(variable octopus.Variable, resourceName string) *string {
	if variable.IsSensitive {
		value := "${var." + resourceName + "}"
		return &value
	}

	return nil
}

func (c *VariableSetConverter) convertValue(variable octopus.Variable, resourceName string) *string {
	if !variable.IsSensitive {
		value := "${var." + resourceName + "}"
		return &value
	}

	return nil
}

func (c *VariableSetConverter) convertPrompt(prompt octopus.Prompt) *terraform.TerraformProjectVariablePrompt {
	if strutil.EmptyIfNil(prompt.Label) != "" || strutil.EmptyIfNil(prompt.Description) != "" {
		return &terraform.TerraformProjectVariablePrompt{
			Description:     prompt.Description,
			Label:           prompt.Label,
			IsRequired:      prompt.Required,
			DisplaySettings: c.convertDisplaySettings(prompt),
		}
	}

	return nil
}

func (c *VariableSetConverter) convertDisplaySettings(prompt octopus.Prompt) *terraform.TerraformProjectVariableDisplay {
	if prompt.DisplaySettings != nil {

		display := terraform.TerraformProjectVariableDisplay{}
		if controlType, ok := prompt.DisplaySettings["Octopus.ControlType"]; ok {
			display.ControlType = &controlType
		}

		selectOptionsSlice := []terraform.TerraformProjectVariableDisplaySelectOption{}
		if selectOptions, ok := prompt.DisplaySettings["Octopus.SelectOptions"]; ok {
			for _, o := range strings.Split(selectOptions, "\n") {
				split := strings.Split(o, "|")
				if len(split) == 2 {
					selectOptionsSlice = append(
						selectOptionsSlice,
						terraform.TerraformProjectVariableDisplaySelectOption{
							DisplayName: split[0],
							Value:       split[1],
						})
				}
			}
		}
		display.SelectOption = &selectOptionsSlice

		return &display
	}

	return nil
}

func (c *VariableSetConverter) convertEnvironmentsToIds() {
	if c.ExcludeVariableEnvironmentScopes == nil {
		c.excludeVariableEnvironmentScopesIds = []string{}
	} else {
		c.excludeVariableEnvironmentScopesIds = lo.FilterMap(c.ExcludeVariableEnvironmentScopes, func(envName string, index int) (string, bool) {

			// for each input environment name, convert it to an ID
			environments := octopus.GeneralCollection[octopus.Environment]{}
			err := c.Client.GetAllResources("Environments", &environments)
			if err == nil {
				// partial matches can have false positives, so do a second filter to do an exact match
				filteredList := lo.FilterMap(environments.Items, func(env octopus.Environment, index int) (string, bool) {
					if env.Name == envName {
						return env.Id, true
					}

					return "", false
				})

				// return the environment id
				if len(filteredList) != 0 {
					return filteredList[0], true
				}
			}

			// no match found
			return "", false
		})
	}
}

func (c *VariableSetConverter) filterEnvironmentScope(envs []string) []string {
	if envs == nil {
		return []string{}
	}

	return lo.Filter(envs, func(env string, i int) bool {
		if c.excludeVariableEnvironmentScopesIds != nil && slices.Index(c.excludeVariableEnvironmentScopesIds, env) != -1 {
			return false
		}

		return true
	})
}

func (c *VariableSetConverter) convertScope(variable octopus.Variable, dependencies *data.ResourceDetailsCollection) *terraform.TerraformProjectVariableScope {
	filteredEnvironments := c.filterEnvironmentScope(variable.Scope.Environment)

	// Removing all environment scoping may not have been the intention
	if len(filteredEnvironments) == 0 && len(variable.Scope.Environment) != 0 {
		zap.L().Warn("WARNING: Variable " + variable.Name + " removed all environment scopes.")
	}

	actions := dependencies.GetResources("Actions", variable.Scope.Action...)
	channels := dependencies.GetResources("Channels", variable.Scope.Channel...)
	environments := dependencies.GetResources("Environments", filteredEnvironments...)
	machines := dependencies.GetResources("Machines", variable.Scope.Machine...)

	if len(actions) != 0 ||
		len(channels) != 0 ||
		len(environments) != 0 ||
		len(machines) != 0 ||
		len(variable.Scope.Role) != 0 ||
		len(variable.Scope.TenantTag) != 0 {

		return &terraform.TerraformProjectVariableScope{
			Actions:      actions,
			Channels:     channels,
			Environments: environments,
			Machines:     machines,
			Roles:        variable.Scope.Role,
			TenantTags:   variable.Scope.TenantTag,
		}
	}

	return nil

}

func (c *VariableSetConverter) exportAccounts(recursive bool, lookup bool, stateless bool, value *string, dependencies *data.ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	for _, account := range regexes.AccountRegex.FindAllString(*value, -1) {
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

	return nil
}

func (c *VariableSetConverter) getAccount(value *string, dependencies *data.ResourceDetailsCollection) *string {
	if value == nil {
		return nil
	}

	retValue := *value

	for _, account := range regexes.AccountRegex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, account, dependencies.GetResource("Accounts", account))
	}

	return &retValue
}

func (c *VariableSetConverter) exportFeeds(recursive bool, lookup bool, stateless bool, value *string, dependencies *data.ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, feed := range regexes.FeedRegex.FindAllString(*value, -1) {
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

	return nil
}

func (c *VariableSetConverter) getFeeds(value *string, dependencies *data.ResourceDetailsCollection) *string {
	if value == nil {
		return nil
	}

	retValue := *value
	for _, account := range regexes.FeedRegex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, account, dependencies.GetResource("Feeds", account))
	}

	return &retValue
}

func (c *VariableSetConverter) exportAzureCloudServiceTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.AzureCloudServiceTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.AzureCloudServiceTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.AzureCloudServiceTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportAzureServiceFabricTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.AzureServiceFabricTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.AzureServiceFabricTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.AzureServiceFabricTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportAzureWebAppTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.AzureWebAppTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.AzureWebAppTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.AzureWebAppTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportCloudRegionTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.CloudRegionTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.CloudRegionTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.CloudRegionTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportKubernetesTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.KubernetesTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.KubernetesTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.KubernetesTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportListeningTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.ListeningTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.ListeningTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.ListeningTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportOfflineDropTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.OfflineDropTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.OfflineDropTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.OfflineDropTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportPollingTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.PollingTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.PollingTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.PollingTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportSshTargets(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			if stateless {
				err = c.SshTargetConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.SshTargetConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.SshTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportEnvironments(recursive bool, lookup bool, stateless bool, variable *octopus.Variable, dependencies *data.ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, e := range c.filterEnvironmentScope(variable.Scope.Environment) {
		var err error
		if recursive {
			if stateless {
				err = c.EnvironmentConverter.ToHclStatelessById(e, dependencies)
			} else {
				err = c.EnvironmentConverter.ToHclById(e, dependencies)
			}
		} else if lookup {
			err = c.EnvironmentConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) exportCertificates(recursive bool, lookup bool, stateless bool, value *string, dependencies *data.ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	if recursive && lookup {
		return errors.New("one, and only one, of recursive and lookup can be true")
	}

	for _, cert := range regexes.CertificatesRegex.FindAllString(*value, -1) {
		var err error
		if recursive {
			if stateless {
				err = c.CertificateConverter.ToHclStatelessById(cert, dependencies)
			} else {
				err = c.CertificateConverter.ToHclById(cert, dependencies)
			}
		} else if lookup {
			err = c.CertificateConverter.ToHclLookupById(cert, dependencies)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) getCertificates(value *string, dependencies *data.ResourceDetailsCollection) *string {
	if value == nil {
		return nil
	}

	retValue := *value
	for _, cert := range regexes.CertificatesRegex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, cert, dependencies.GetResource("Certificates", cert))
	}

	return &retValue
}

func (c *VariableSetConverter) exportWorkerPools(recursive bool, lookup bool, stateless bool, value *string, dependencies *data.ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	for _, pool := range regexes.WorkerPoolsRegex.FindAllString(*value, -1) {
		var err error
		if recursive {
			if stateless {
				err = c.WorkerPoolConverter.ToHclStatelessById(pool, dependencies)
			} else {
				err = c.WorkerPoolConverter.ToHclById(pool, dependencies)
			}
		} else if lookup {
			err = c.WorkerPoolConverter.ToHclLookupById(pool, dependencies)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *VariableSetConverter) getWorkerPools(value *string, dependencies *data.ResourceDetailsCollection) *string {
	if len(strutil.EmptyIfNil(value)) == 0 {
		return nil
	}

	retValue := *value
	for _, cert := range regexes.WorkerPoolsRegex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, cert, dependencies.GetResource("WorkerPools", cert))
	}

	return &retValue
}

// addTagSetDependencies finds the tag sets that contains the tags associated with a tenant. These dependencies are
// captured, as Terraform has no other way to map the dependency between a tagset and a tenant.
func (c *VariableSetConverter) addTagSetDependencies(variable octopus.Variable, recursive bool, dependencies *data.ResourceDetailsCollection) (map[string][]string, error) {
	collection := octopus.GeneralCollection[octopus.TagSet]{}
	err := c.Client.GetAllResources("TagSets", &collection)

	if err != nil {
		return nil, err
	}

	terraformDependencies := map[string][]string{}

	for _, tagSet := range collection.Items {
		for _, tag := range tagSet.Tags {
			for _, tenantTag := range variable.Scope.TenantTag {
				if tag.CanonicalTagName == tenantTag {

					if !slices.Contains(terraformDependencies["TagSets"], tagSet.Id) {
						terraformDependencies["TagSets"] = append(terraformDependencies["TagSets"], tagSet.Id)
					}

					if !slices.Contains(terraformDependencies["Tags"], tag.Id) {
						terraformDependencies["Tags"] = append(terraformDependencies["Tags"], tag.Id)
					}

					if recursive {
						err = c.TagSetConverter.ToHclByResource(tagSet, dependencies)

						if err != nil {
							return nil, err
						}
					}
				}
			}
		}
	}

	return terraformDependencies, nil
}
