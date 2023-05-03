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
	"k8s.io/utils/strings/slices"
	"regexp"
	"strings"
)

// VariableSetConverter exports variable sets.
// Note that we only access variable sets as dependencies of other resources, like project variables or
// library variable sets. There is no global collection or all endpoint that we can use to dump variables
// in bulk.
type VariableSetConverter struct {
	Client                            client.OctopusClient
	ChannelConverter                  ConverterByProjectIdWithTerraDependencies
	EnvironmentConverter              ConverterAndLookupById
	TagSetConverter                   TagSetConverter
	AzureCloudServiceTargetConverter  ConverterAndLookupById
	AzureServiceFabricTargetConverter ConverterAndLookupById
	AzureWebAppTargetConverter        ConverterAndLookupById
	CloudRegionTargetConverter        ConverterAndLookupById
	KubernetesTargetConverter         ConverterAndLookupById
	ListeningTargetConverter          ConverterAndLookupById
	OfflineDropTargetConverter        ConverterAndLookupById
	PollingTargetConverter            ConverterAndLookupById
	SshTargetConverter                ConverterAndLookupById
	AccountConverter                  ConverterAndLookupById
	FeedConverter                     ConverterAndLookupById
	CertificateConverter              ConverterAndLookupById
	WorkerPoolConverter               ConverterAndLookupById
	IgnoreCacManagedValues            bool
	DefaultSecretVariableValues       bool
}

// ToHclByProjectIdAndName is called when returning variables from projects. This is because the variable set ID
// defined on a CaC enabled project is not available from the global /variablesets endpoint, and can only be
// accessed from the project resource.
func (c VariableSetConverter) ToHclByProjectIdAndName(projectId string, parentName string, parentLookup string, dependencies *ResourceDetailsCollection) error {
	if projectId == "" {
		return nil
	}

	variables := octopus.VariableSet{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &variables)

	if err != nil {
		return err
	}

	project := octopus.Project{}
	_, err = c.Client.GetResourceById("Projects", projectId, &project)

	if err != nil {
		return err
	}

	ignoreSecrets := project.HasCacConfigured() && c.IgnoreCacManagedValues

	return c.toHcl(variables, true, false, ignoreSecrets, parentName, parentLookup, dependencies)
}

func (c VariableSetConverter) ToHclLookupByProjectIdAndName(projectId string, parentName string, parentLookup string, dependencies *ResourceDetailsCollection) error {
	if projectId == "" {
		return nil
	}

	variables := octopus.VariableSet{}
	_, err := c.Client.GetResource(c.GetGroupResourceType(projectId), &variables)

	if err != nil {
		return err
	}

	project := octopus.Project{}
	_, err = c.Client.GetResourceById("Projects", projectId, &project)

	if err != nil {
		return err
	}

	ignoreSecrets := project.HasCacConfigured() && c.IgnoreCacManagedValues

	return c.toHcl(variables, false, true, ignoreSecrets, parentName, parentLookup, dependencies)
}

func (c VariableSetConverter) ToHclByIdAndName(id string, parentName string, parentLookup string, dependencies *ResourceDetailsCollection) error {
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

	return c.toHcl(resource, true, false, false, parentName, parentLookup, dependencies)
}

// ToHclLookupByIdAndName exports the variable set as a complete resource, but will reference external resources like accounts,
// feeds, worker pools, certificates, environments, and targets as data source lookups.
func (c VariableSetConverter) ToHclLookupByIdAndName(id string, parentName string, parentLookup string, dependencies *ResourceDetailsCollection) error {
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

	return c.toHcl(resource, false, true, false, parentName, parentLookup, dependencies)
}

func (c VariableSetConverter) toHcl(resource octopus.VariableSet, recursive bool, lookup bool, ignoreSecrets bool, parentName string, parentLookup string, dependencies *ResourceDetailsCollection) error {
	if recursive {
		c.exportChildDependencies(resource, dependencies)
	}

	nameCount := map[string]int{}
	for _, v := range resource.Variables {
		// Do not export regular variables if ignoring cac managed values
		if ignoreSecrets && !v.IsSensitive {
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
		thisResource := ResourceDetails{}

		resourceName := sanitizer.SanitizeName(parentName) + "_" + sanitizer.SanitizeName(v.Name) + "_" + fmt.Sprint(nameCount[v.Name])

		// Export linked accounts
		err := c.exportAccounts(recursive, lookup, v.Value, dependencies)
		if err != nil {
			return err
		}

		// Export linked feeds
		err = c.exportFeeds(recursive, lookup, v.Value, dependencies)
		if err != nil {
			return err
		}

		// Export linked worker pools
		err = c.exportWorkerPools(recursive, lookup, v.Value, dependencies)
		if err != nil {
			return err
		}

		// Export linked certificates
		err = c.exportCertificates(recursive, lookup, v.Value, dependencies)
		if err != nil {
			return err
		}

		// Export linked environments
		err = c.exportEnvironments(recursive, lookup, &v, dependencies)
		if err != nil {
			return err
		}

		// Export azure cloud service targets
		err = c.exportAzureCloudServiceTargets(recursive, lookup, &v, dependencies)
		if err != nil {
			return err
		}

		// Export azure service fabric targets
		err = c.exportAzureServiceFabricTargets(recursive, lookup, &v, dependencies)
		if err != nil {
			return err
		}

		// Export azure web app targets
		err = c.exportAzureWebAppTargets(recursive, lookup, &v, dependencies)
		if err != nil {
			return err
		}

		// Export azure web app targets
		err = c.exportCloudRegionTargets(recursive, lookup, &v, dependencies)
		if err != nil {
			return err
		}

		// Export kubernetes targets
		err = c.exportKubernetesTargets(recursive, lookup, &v, dependencies)
		if err != nil {
			return err
		}

		// Export listening targets
		err = c.exportListeningTargets(recursive, lookup, &v, dependencies)
		if err != nil {
			return err
		}

		// Export listening targets
		err = c.exportOfflineDropTargets(recursive, lookup, &v, dependencies)
		if err != nil {
			return err
		}

		// Export polling targets
		err = c.exportPollingTargets(recursive, lookup, &v, dependencies)
		if err != nil {
			return err
		}

		// Export polling targets
		err = c.exportSshTargets(recursive, lookup, &v, dependencies)
		if err != nil {
			return err
		}

		tagSetDependencies, err := c.addTagSetDependencies(v, recursive, dependencies)

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
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${octopusdeploy_variable." + resourceName + ".id}"
		thisResource.ToHcl = func() (string, error) {

			// Replace anything that looks like an octopus resource reference
			value := c.getAccount(v.Value, dependencies)
			value = c.getFeeds(value, dependencies)
			value = c.getCertificates(value, dependencies)
			value = c.getWorkerPools(value, dependencies)

			terraformResource := terraform.TerraformProjectVariable{
				Name:           resourceName,
				Type:           "octopusdeploy_variable",
				OwnerId:        parentLookup,
				Value:          value,
				ResourceName:   v.Name,
				ResourceType:   v.Type,
				Description:    v.Description,
				SensitiveValue: c.convertSecretValue(v, resourceName),
				IsSensitive:    v.IsSensitive,
				Prompt:         c.convertPrompt(v.Prompt),
				Scope:          c.convertScope(v.Scope, dependencies),
			}

			if v.IsSensitive {
				var defaultValue *string = nil

				if c.DefaultSecretVariableValues {
					defaultValueLookup := "#{" + v.Name + "}"
					defaultValue = &defaultValueLookup
				}

				secretVariableResource := terraform.TerraformVariable{
					Name:        resourceName,
					Type:        "string",
					Nullable:    false,
					Sensitive:   true,
					Description: "The secret variable value associated with the variable " + v.Name,
					Default:     defaultValue,
				}

				block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
				hcl.WriteUnquotedAttribute(block, "type", "string")

				file.Body().AppendBlock(block)
			} else if v.Type == "String" {
				// Use a second terraform variable to allow the octopus variable to be defined at apply time.
				// Note this only applies to string variables, as other types likely reference resources
				// that are being created by terraform, and these dynamic values can not be used as default
				// variable values.
				terraformResource.Value = c.convertValue(v, resourceName)
				regularVariable := terraform.TerraformVariable{
					Name:        resourceName,
					Type:        "string",
					Nullable:    false,
					Sensitive:   false,
					Description: "The value associated with the variable " + v.Name,
					Default:     value,
				}

				block := gohcl.EncodeAsBlock(regularVariable, "variable")
				hcl.WriteUnquotedAttribute(block, "type", "string")
				// variable lookups need to be raw expressions
				if hcl.IsInterpolation(strutil.EmptyIfNil(value)) {
					hcl.WriteUnquotedAttribute(block, "default", strutil.EmptyIfNil(value))
				}
				file.Body().AppendBlock(block)
			}

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			// Explicitly describe the dependency between a target and a tag set
			dependsOn := []string{}
			for resourceType, terraformDependencies := range tagSetDependencies {
				for _, terraformDependency := range terraformDependencies {
					dependency := dependencies.GetResource(resourceType, terraformDependency)
					dependency = hcl.RemoveId(hcl.RemoveInterpolation(dependency))
					dependsOn = append(dependsOn, dependency)
				}
			}
			hcl.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(dependsOn[:], ",")+"]")

			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c VariableSetConverter) GetResourceType() string {
	return "Variables"
}

func (c VariableSetConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/Variables"
}

func (c VariableSetConverter) convertSecretValue(variable octopus.Variable, resourceName string) *string {
	if variable.IsSensitive {
		value := "${var." + resourceName + "}"
		return &value
	}

	return nil
}

func (c VariableSetConverter) convertValue(variable octopus.Variable, resourceName string) *string {
	if !variable.IsSensitive {
		value := "${var." + resourceName + "}"
		return &value
	}

	return nil
}

func (c VariableSetConverter) convertPrompt(prompt octopus.Prompt) *terraform.TerraformProjectVariablePrompt {
	if prompt.Label != nil || prompt.Description != nil {
		return &terraform.TerraformProjectVariablePrompt{
			Description: prompt.Description,
			Label:       prompt.Label,
			IsRequired:  prompt.Required,
		}
	}

	return nil
}

func (c VariableSetConverter) convertScope(prompt octopus.Scope, dependencies *ResourceDetailsCollection) *terraform.TerraformProjectVariableScope {
	return &terraform.TerraformProjectVariableScope{
		Actions:      dependencies.GetResources("Actions", prompt.Action...),
		Channels:     dependencies.GetResources("Channels", prompt.Channel...),
		Environments: dependencies.GetResources("Environments", prompt.Environment...),
		Machines:     dependencies.GetResources("Machines", prompt.Machine...),
		Roles:        prompt.Role,
		TenantTags:   prompt.TenantTag,
	}

}

func (c VariableSetConverter) exportAccounts(recursive bool, lookup bool, value *string, dependencies *ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	accountRegex := regexp.MustCompile("Accounts-\\d+")
	for _, account := range accountRegex.FindAllString(*value, -1) {
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

	return nil
}

func (c VariableSetConverter) getAccount(value *string, dependencies *ResourceDetailsCollection) *string {
	if value == nil {
		return nil
	}

	retValue := *value
	accountRegex := regexp.MustCompile("Accounts-\\d+")
	for _, account := range accountRegex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, account, dependencies.GetResource("Accounts", account))
	}

	return &retValue
}

func (c VariableSetConverter) exportFeeds(recursive bool, lookup bool, value *string, dependencies *ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	feedRegex := regexp.MustCompile("Feeds-\\d+")
	for _, feed := range feedRegex.FindAllString(*value, -1) {
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

	return nil
}

func (c VariableSetConverter) getFeeds(value *string, dependencies *ResourceDetailsCollection) *string {
	if value == nil {
		return nil
	}

	retValue := *value
	regex := regexp.MustCompile("Feeds-\\d+")
	for _, account := range regex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, account, dependencies.GetResource("Feeds", account))
	}

	return &retValue
}

func (c VariableSetConverter) exportAzureCloudServiceTargets(recursive bool, lookup bool, variable *octopus.Variable, dependencies *ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			err = c.AzureCloudServiceTargetConverter.ToHclById(e, dependencies)
		} else if lookup {
			err = c.AzureCloudServiceTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c VariableSetConverter) exportAzureServiceFabricTargets(recursive bool, lookup bool, variable *octopus.Variable, dependencies *ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			err = c.AzureServiceFabricTargetConverter.ToHclById(e, dependencies)
		} else if lookup {
			err = c.AzureServiceFabricTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c VariableSetConverter) exportAzureWebAppTargets(recursive bool, lookup bool, variable *octopus.Variable, dependencies *ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			err = c.AzureWebAppTargetConverter.ToHclById(e, dependencies)
		} else if lookup {
			err = c.AzureWebAppTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c VariableSetConverter) exportCloudRegionTargets(recursive bool, lookup bool, variable *octopus.Variable, dependencies *ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			err = c.CloudRegionTargetConverter.ToHclById(e, dependencies)
		} else if lookup {
			err = c.CloudRegionTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c VariableSetConverter) exportKubernetesTargets(recursive bool, lookup bool, variable *octopus.Variable, dependencies *ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			err = c.KubernetesTargetConverter.ToHclById(e, dependencies)
		} else if lookup {
			err = c.KubernetesTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c VariableSetConverter) exportListeningTargets(recursive bool, lookup bool, variable *octopus.Variable, dependencies *ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			err = c.ListeningTargetConverter.ToHclById(e, dependencies)
		} else if lookup {
			err = c.ListeningTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c VariableSetConverter) exportOfflineDropTargets(recursive bool, lookup bool, variable *octopus.Variable, dependencies *ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			err = c.OfflineDropTargetConverter.ToHclById(e, dependencies)
		} else if lookup {
			err = c.OfflineDropTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c VariableSetConverter) exportPollingTargets(recursive bool, lookup bool, variable *octopus.Variable, dependencies *ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			err = c.PollingTargetConverter.ToHclById(e, dependencies)
		} else if lookup {
			err = c.PollingTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c VariableSetConverter) exportSshTargets(recursive bool, lookup bool, variable *octopus.Variable, dependencies *ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	for _, e := range variable.Scope.Machine {
		var err error
		if recursive {
			err = c.SshTargetConverter.ToHclById(e, dependencies)
		} else if lookup {
			err = c.SshTargetConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c VariableSetConverter) exportEnvironments(recursive bool, lookup bool, variable *octopus.Variable, dependencies *ResourceDetailsCollection) error {
	if variable == nil {
		return nil
	}

	for _, e := range variable.Scope.Environment {
		var err error
		if recursive {
			err = c.EnvironmentConverter.ToHclById(e, dependencies)
		} else if lookup {
			err = c.EnvironmentConverter.ToHclLookupById(e, dependencies)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c VariableSetConverter) exportCertificates(recursive bool, lookup bool, value *string, dependencies *ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	regex := regexp.MustCompile("Certificates-\\d+")
	for _, cert := range regex.FindAllString(*value, -1) {
		var err error
		if recursive {
			err = c.CertificateConverter.ToHclById(cert, dependencies)
		} else if lookup {
			err = c.CertificateConverter.ToHclLookupById(cert, dependencies)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (c VariableSetConverter) getCertificates(value *string, dependencies *ResourceDetailsCollection) *string {
	if value == nil {
		return nil
	}

	retValue := *value
	regex := regexp.MustCompile("Certificates-\\d+")
	for _, cert := range regex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, cert, dependencies.GetResource("Certificates", cert))
	}

	return &retValue
}

func (c VariableSetConverter) exportWorkerPools(recursive bool, lookup bool, value *string, dependencies *ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	regex := regexp.MustCompile("WorkerPools-\\d+")
	for _, pool := range regex.FindAllString(*value, -1) {
		var err error
		if recursive {
			err = c.WorkerPoolConverter.ToHclById(pool, dependencies)
		} else if lookup {
			err = c.WorkerPoolConverter.ToHclLookupById(pool, dependencies)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (c VariableSetConverter) getWorkerPools(value *string, dependencies *ResourceDetailsCollection) *string {
	if value == nil {
		return nil
	}

	retValue := *value
	regex := regexp.MustCompile("WorkerPools-\\d+")
	for _, cert := range regex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, cert, dependencies.GetResource("WorkerPools", cert))
	}

	return &retValue
}

func (c VariableSetConverter) exportChildDependencies(variableSet octopus.VariableSet, dependencies *ResourceDetailsCollection) error {
	for _, v := range variableSet.Variables {
		for _, e := range v.Scope.Environment {
			err := c.EnvironmentConverter.ToHclById(e, dependencies)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// addTagSetDependencies finds the tag sets that contains the tags associated with a tenant. These dependencies are
// captured, as Terraform has no other way to map the dependency between a tagset and a tenant.
func (c VariableSetConverter) addTagSetDependencies(variable octopus.Variable, recursive bool, dependencies *ResourceDetailsCollection) (map[string][]string, error) {
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
