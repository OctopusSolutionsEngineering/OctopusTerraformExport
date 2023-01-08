package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type EnvironmentConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
}

func (c EnvironmentConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.Environment]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	for _, environment := range collection.Items {
		environmentName := "environment_" + util.SanitizeName(environment.Name)

		terraformResource := terraform.TerraformEnvironment{
			Type:                       "octopusdeploy_environment",
			Name:                       environmentName,
			SpaceId:                    nil,
			ResourceName:               environment.Name,
			Description:                environment.Description,
			AllowDynamicInfrastructure: environment.AllowDynamicInfrastructure,
			UseGuidedFailure:           environment.UseGuidedFailure,
			SortOrder:                  0,
			JiraExtensionSettings: &terraform.TerraformJiraExtensionSettings{
				EnvironmentType: c.GetJiraExtensionSettings(environment),
			},
			JiraServiceManagementExtensionSettings: &terraform.TerraformJiraServiceManagementExtensionSettings{
				IsEnabled: c.GetJiraServiceManagementExtensionSettings(environment),
			},
			ServicenowExtensionSettings: &terraform.TerraformServicenowExtensionSettings{
				IsEnabled: c.GetServiceNowChangeControlled(environment),
			},
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		results["space_population/environment_"+environmentName+".tf"] = string(file.Bytes())
		resultsMap[environment.Id] = "${octopusdeploy_environment." + environmentName + ".id}"
	}

	return results, resultsMap, nil
}

func (c EnvironmentConverter) GetServiceNowChangeControlled(env octopus.Environment) bool {
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

func (c EnvironmentConverter) GetJiraServiceManagementExtensionSettings(env octopus.Environment) bool {
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

func (c EnvironmentConverter) GetJiraExtensionSettings(env octopus.Environment) string {
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
