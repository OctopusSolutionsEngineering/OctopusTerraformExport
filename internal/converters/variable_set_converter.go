package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
	"regexp"
	"strings"
)

// VariableSetConverter exports variable sets.
// Note that we only access variable sets as dependencies of other resources, like project variables or
// library variable sets. There is no global collection or all endpoint that we can use to dump variables
// in bulk.
type VariableSetConverter struct {
	Client client.OctopusClient
}

func (c VariableSetConverter) ToHclById(id string, recursive bool, parentName string, parentLookup string, dependencies *ResourceDetailsCollection) error {
	resource := octopus.VariableSet{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, recursive, parentName, parentLookup, dependencies)
}

func (c VariableSetConverter) toHcl(resource octopus.VariableSet, recursive bool, parentName string, parentLookup string, dependencies *ResourceDetailsCollection) error {
	file := hclwrite.NewEmptyFile()

	for _, v := range resource.Variables {
		thisResource := ResourceDetails{}

		resourceName := parentName + "_" + util.SanitizeName(v.Name)

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

			terraformResource := terraform.TerraformProjectVariable{
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
			}

			if v.IsSensitive {
				secretVariableResource := terraform.TerraformVariable{
					Name:        parentName,
					Type:        "string",
					Nullable:    false,
					Sensitive:   true,
					Description: "The secret variable value associated with the variable " + v.Name,
				}

				block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
				util.WriteUnquotedAttribute(block, "type", "string")
				file.Body().AppendBlock(block)
			}

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}
		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c VariableSetConverter) GetResourceType() string {
	return "Variables"
}

func (c VariableSetConverter) convertSecretValue(variable octopus.Variable, parentName string) *string {
	if variable.IsSensitive {
		value := "${var." + parentName + "}"
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
