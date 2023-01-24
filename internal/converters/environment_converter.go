package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/sanitizer"
)

type EnvironmentConverter struct {
	Client client.OctopusClient
}

func (c EnvironmentConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Environment]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c EnvironmentConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if dependencies.HasResource(c.GetResourceType(), id) {
		return nil
	}

	environment := octopus.Environment{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &environment)

	if err != nil {
		return err
	}

	return c.toHcl(environment, true, dependencies)
}

func (c EnvironmentConverter) toHcl(environment octopus.Environment, recursive bool, dependencies *ResourceDetailsCollection) error {
	resourceName := "environment_" + sanitizer.SanitizeName(environment.Name)

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

func (c EnvironmentConverter) getServiceNowChangeControlled(env octopus.Environment) bool {
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

func (c EnvironmentConverter) getJiraServiceManagementExtensionSettings(env octopus.Environment) bool {
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

func (c EnvironmentConverter) getJiraExtensionSettings(env octopus.Environment) string {
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

func (c EnvironmentConverter) GetResourceType() string {
	return "Environments"
}
