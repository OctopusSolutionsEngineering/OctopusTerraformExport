package singleconverter

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

type SingleVariableSetConverter struct {
	Client client.OctopusClient
}

func (c SingleVariableSetConverter) ToHclById(id string, parentName string, parentLookup string, dependencies *ResourceDetailsCollection) error {
	resource := octopus.VariableSet{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	file := hclwrite.NewEmptyFile()

	for _, v := range resource.Variables {
		thisResource := ResourceDetails{}

		resourceName := parentName + "_" + util.SanitizeName(v.Name)

		// Export linked accounts
		err = c.exportAccounts(v.Value, dependencies)
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

		thisResource.FileName = "space_population/project_variable_" + resourceName + ".tf"
		thisResource.Id = v.Id
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = "${octopusdeploy_variable." + resourceName + ".id}"
		thisResource.ToHcl = func() (string, error) {

			// Replace anything that looks like an octopus resource reference
			value := c.getAccount(v.Value, dependencies)
			value = c.getFeeds(value, dependencies)
			value = c.getCertificates(value, dependencies)

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

func (c SingleVariableSetConverter) GetResourceType() string {
	return "Variables"
}

func (c SingleVariableSetConverter) convertSecretValue(variable octopus.Variable, parentName string) *string {
	if variable.IsSensitive {
		value := "${var." + parentName + "}"
		return &value
	}

	return nil
}

func (c SingleVariableSetConverter) convertPrompt(prompt octopus.Prompt) *terraform.TerraformProjectVariablePrompt {
	if prompt.Label != nil || prompt.Description != nil {
		return &terraform.TerraformProjectVariablePrompt{
			Description: prompt.Description,
			Label:       prompt.Label,
			IsRequired:  prompt.Required,
		}
	}

	return nil
}

func (c SingleVariableSetConverter) exportAccounts(value *string, dependencies *ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	accountRegex, _ := regexp.Compile("Accounts-\\d+")
	for _, account := range accountRegex.FindAllString(*value, -1) {
		err := SingleAccountConverter{
			Client: c.Client,
		}.ToHclById(account, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c SingleVariableSetConverter) getAccount(value *string, dependencies *ResourceDetailsCollection) *string {
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

func (c SingleVariableSetConverter) exportFeeds(value *string, dependencies *ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	feedRegex, _ := regexp.Compile("Feeds-\\d+")
	for _, account := range feedRegex.FindAllString(*value, -1) {
		err := SingleFeedConverter{
			Client: c.Client,
		}.ToHclById(account, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c SingleVariableSetConverter) getFeeds(value *string, dependencies *ResourceDetailsCollection) *string {
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

func (c SingleVariableSetConverter) exportCertificates(value *string, dependencies *ResourceDetailsCollection) error {
	if value == nil {
		return nil
	}

	regex, _ := regexp.Compile("Certificates-\\d+")
	for _, cert := range regex.FindAllString(*value, -1) {
		err := SingleCertificateConverter{
			Client: c.Client,
		}.ToHclById(cert, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c SingleVariableSetConverter) getCertificates(value *string, dependencies *ResourceDetailsCollection) *string {
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
