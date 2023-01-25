package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
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
	EnvironmentConverter              ConverterById
	TagSetConverter                   TagSetConverter
	AzureCloudServiceTargetConverter  ConverterById
	AzureServiceFabricTargetConverter ConverterById
	AzureWebAppTargetConverter        ConverterById
	CloudRegionTargetConverter        ConverterById
	KubernetesTargetConverter         ConverterById
	ListeningTargetConverter          ConverterById
	OfflineDropTargetConverter        ConverterById
	PollingTargetConverter            ConverterById
	SshTargetConverter                ConverterById
}

func (c VariableSetConverter) ToHclByIdAndName(id string, parentName string, parentLookup string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.VariableSet{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, parentName, parentLookup, dependencies)
}

func (c VariableSetConverter) toHcl(resource octopus2.VariableSet, recursive bool, parentName string, parentLookup string, dependencies *ResourceDetailsCollection) error {
	if recursive {
		c.exportChildDependencies(resource, dependencies)
	}

	for i, v := range resource.Variables {
		v := v
		file := hclwrite.NewEmptyFile()
		thisResource := ResourceDetails{}

		resourceName := sanitizer.SanitizeName(parentName) + "_" + sanitizer.SanitizeName(v.Name) + "_" + fmt.Sprint(i)

		if recursive {
			// Export linked accounts
			err := c.exportAccounts(v.Value, dependencies)
			if err != nil {
				return err
			}

			// Export linked feeds
			err = c.exportFeeds(v.Value, dependencies)
			if err != nil {
				return err
			}

			// Export linked certificates
			err = c.exportCertificates(v.Value, dependencies)
			if err != nil {
				return err
			}

			// Export linked worker pools
			err = c.exportWorkerPools(v.Value, dependencies)
			if err != nil {
				return err
			}

			// Export linked environments
			for _, e := range v.Scope.Environment {
				err = c.EnvironmentConverter.ToHclById(e, dependencies)
				if err != nil {
					return err
				}
			}

			// Export linked targets
			for _, m := range v.Scope.Machine {
				err = c.AzureCloudServiceTargetConverter.ToHclById(m, dependencies)
				if err != nil {
					return err
				}

				err = c.AzureServiceFabricTargetConverter.ToHclById(m, dependencies)
				if err != nil {
					return err
				}

				err = c.AzureWebAppTargetConverter.ToHclById(m, dependencies)
				if err != nil {
					return err
				}

				err = c.CloudRegionTargetConverter.ToHclById(m, dependencies)
				if err != nil {
					return err
				}

				err = c.KubernetesTargetConverter.ToHclById(m, dependencies)
				if err != nil {
					return err
				}

				err = c.ListeningTargetConverter.ToHclById(m, dependencies)
				if err != nil {
					return err
				}

				err = c.OfflineDropTargetConverter.ToHclById(m, dependencies)
				if err != nil {
					return err
				}

				err = c.PollingTargetConverter.ToHclById(m, dependencies)
				if err != nil {
					return err
				}

				err = c.SshTargetConverter.ToHclById(m, dependencies)
				if err != nil {
					return err
				}
			}
		}

		tagSetDependencies, err := c.addTagSetDependencies(v, recursive, dependencies)

		if err != nil {
			return err
		}

		thisResource.FileName = "space_population/project_variable_" + resourceName + ".tf"
		thisResource.Id = v.Id
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${octopusdeploy_variable." + resourceName + ".id}"
		thisResource.ToHcl = func() (string, error) {

			// Replace anything that looks like an octopus resource reference
			value := c.getAccount(v.Value, dependencies)
			value = c.getFeeds(value, dependencies)
			value = c.getCertificates(value, dependencies)
			value = c.getWorkerPools(value, dependencies)

			terraformResource := terraform2.TerraformProjectVariable{
				Name:           resourceName,
				Type:           "octopusdeploy_variable",
				OwnerId:        parentLookup,
				Value:          value,
				ResourceName:   v.Name,
				ResourceType:   v.Type,
				Description:    v.Description,
				SensitiveValue: c.convertSecretValue(v, parentName),
				IsSensitive:    v.IsSensitive,
				Prompt:         c.convertPrompt(v.Prompt),
				Scope:          c.convertScope(v.Scope, dependencies),
			}

			if v.IsSensitive {
				secretVariableResource := terraform2.TerraformVariable{
					Name:        sanitizer.SanitizeName(parentName),
					Type:        "string",
					Nullable:    false,
					Sensitive:   true,
					Description: "The secret variable value associated with the variable " + v.Name,
				}

				block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
				hcl.WriteUnquotedAttribute(block, "type", "string")
				file.Body().AppendBlock(block)
			}

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			// Explicitly describe the dependency between a target and a tag set
			dependsOn := []string{}
			for resourceType, terraformDependencies := range tagSetDependencies {
				for _, terraformDependency := range terraformDependencies {
					dependency := dependencies.GetResource(resourceType, terraformDependency)
					// This is a raw expression, so remove the surrounding brackets
					dependency = strings.Replace(dependency, "${", "", -1)
					dependency = strings.Replace(dependency, ".id}", "", -1)
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

func (c VariableSetConverter) convertSecretValue(variable octopus2.Variable, parentName string) *string {
	if variable.IsSensitive {
		value := "${var." + sanitizer.SanitizeName(parentName) + "}"
		return &value
	}

	return nil
}

func (c VariableSetConverter) convertPrompt(prompt octopus2.Prompt) *terraform2.TerraformProjectVariablePrompt {
	if prompt.Label != nil || prompt.Description != nil {
		return &terraform2.TerraformProjectVariablePrompt{
			Description: prompt.Description,
			Label:       prompt.Label,
			IsRequired:  prompt.Required,
		}
	}

	return nil
}

func (c VariableSetConverter) convertScope(prompt octopus2.Scope, dependencies *ResourceDetailsCollection) *terraform2.TerraformProjectVariableScope {
	return &terraform2.TerraformProjectVariableScope{
		Actions:      dependencies.GetResources("Actions", prompt.Action...),
		Channels:     dependencies.GetResources("Channels", prompt.Channel...),
		Environments: dependencies.GetResources("Environments", prompt.Environment...),
		Machines:     dependencies.GetResources("Machines", prompt.Machine...),
		Roles:        prompt.Role,
		TenantTags:   prompt.TenantTag,
	}

}

func (c VariableSetConverter) exportAccounts(value *string, dependencies *ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	accountRegex, _ := regexp.Compile("Accounts-\\d+")
	for _, account := range accountRegex.FindAllString(*value, -1) {
		err := AccountConverter{
			Client: c.Client,
		}.ToHclById(account, dependencies)

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
	accountRegex, _ := regexp.Compile("Accounts-\\d+")
	for _, account := range accountRegex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, account, dependencies.GetResource("Accounts", account))
	}

	return &retValue
}

func (c VariableSetConverter) exportFeeds(value *string, dependencies *ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	feedRegex, _ := regexp.Compile("Feeds-\\d+")
	for _, account := range feedRegex.FindAllString(*value, -1) {
		err := FeedConverter{
			Client: c.Client,
		}.ToHclById(account, dependencies)

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
	regex, _ := regexp.Compile("Feeds-\\d+")
	for _, account := range regex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, account, dependencies.GetResource("Feeds", account))
	}

	return &retValue
}

func (c VariableSetConverter) exportCertificates(value *string, dependencies *ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	regex, _ := regexp.Compile("Certificates-\\d+")
	for _, cert := range regex.FindAllString(*value, -1) {
		err := CertificateConverter{
			Client: c.Client,
		}.ToHclById(cert, dependencies)

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
	regex, _ := regexp.Compile("Certificates-\\d+")
	for _, cert := range regex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, cert, dependencies.GetResource("Certificates", cert))
	}

	return &retValue
}

func (c VariableSetConverter) exportWorkerPools(value *string, dependencies *ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	regex, _ := regexp.Compile("WorkerPools-\\d+")
	for _, cert := range regex.FindAllString(*value, -1) {
		err := WorkerPoolConverter{
			Client: c.Client,
		}.ToHclById(cert, dependencies)

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
	regex, _ := regexp.Compile("WorkerPools-\\d+")
	for _, cert := range regex.FindAllString(*value, -1) {
		retValue = strings.ReplaceAll(retValue, cert, dependencies.GetResource("WorkerPools", cert))
	}

	return &retValue
}

func (c VariableSetConverter) exportChildDependencies(variableSet octopus2.VariableSet, dependencies *ResourceDetailsCollection) error {
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
func (c VariableSetConverter) addTagSetDependencies(variable octopus2.Variable, recursive bool, dependencies *ResourceDetailsCollection) (map[string][]string, error) {
	collection := octopus2.GeneralCollection[octopus2.TagSet]{}
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
						err = c.TagSetConverter.ToHclByResource(tagSet, recursive, dependencies)

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
