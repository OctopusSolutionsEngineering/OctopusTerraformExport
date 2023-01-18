package singleconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type SingleEnvironmentConverter struct {
	Client client.OctopusClient
}

func (c SingleEnvironmentConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	environment := octopus.Environment{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &environment)

	if err != nil {
		return err
	}

	return c.toHcl(environment, dependencies)
}

func (c SingleEnvironmentConverter) toHcl(environment octopus.Environment, dependencies *ResourceDetailsCollection) error {
	resourceName := "environment_" + util.SanitizeName(environment.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = environment.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_environment." + resourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformEnvironment{
			Type:                       "octopusdeploy_environment",
			Name:                       resourceName,
			SpaceId:                    nil,
			ResourceName:               environment.Name,
			Description:                environment.Description,
			AllowDynamicInfrastructure: environment.AllowDynamicInfrastructure,
			UseGuidedFailure:           environment.UseGuidedFailure,
			SortOrder:                  0,
			JiraExtensionSettings: &terraform.TerraformJiraExtensionSettings{
				EnvironmentType: c.getJiraExtensionSettings(environment),
			},
			JiraServiceManagementExtensionSettings: &terraform.TerraformJiraServiceManagementExtensionSettings{
				IsEnabled: c.getJiraServiceManagementExtensionSettings(environment),
			},
			ServicenowExtensionSettings: &terraform.TerraformServicenowExtensionSettings{
				IsEnabled: c.getServiceNowChangeControlled(environment),
			},
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c SingleEnvironmentConverter) getServiceNowChangeControlled(env octopus.Environment) bool {
	for _, setting := range env.ExtensionSettings {
		if setting.ExtensionId == "servicenow-integration" {
			v, ok := setting.Values["ServiceNowChangeControlled"]
			if ok {
				switch t := v.(type) {
				case bool:
					return t
				}
			}

		}
	}

	return false
}

func (c SingleEnvironmentConverter) getJiraServiceManagementExtensionSettings(env octopus.Environment) bool {
	for _, setting := range env.ExtensionSettings {
		if setting.ExtensionId == "jiraservicemanagement-integration" {
			v, ok := setting.Values["JsmChangeControlled"]
			if ok {
				switch t := v.(type) {
				case bool:
					return t
				}
			}

		}
	}

	return false
}

func (c SingleEnvironmentConverter) getJiraExtensionSettings(env octopus.Environment) string {
	for _, setting := range env.ExtensionSettings {
		if setting.ExtensionId == "jira-integration" {
			v, ok := setting.Values["JiraEnvironmentType"]
			if ok {
				switch t := v.(type) {
				case string:
					return t
				}
			}

		}
	}

	return "unmapped"
}

func (c SingleEnvironmentConverter) GetResourceType() string {
	return "Environments"
}
