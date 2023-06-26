package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
)

type EnvironmentConverter struct {
	Client client.OctopusClient
}

func (c EnvironmentConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.Environment]{}
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
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	environment := octopus2.Environment{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &environment)

	if err != nil {
		return err
	}

	return c.toHcl(environment, true, dependencies)
}

func (c EnvironmentConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	environment := octopus2.Environment{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &environment)

	if err != nil {
		return err
	}

	thisResource := ResourceDetails{}

	resourceName := "environment_" + sanitizer.SanitizeName(environment.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = environment.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data.octopusdeploy_environments." + resourceName + ".environments[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformEnvironmentData{
			Type:        "octopusdeploy_environments",
			Name:        resourceName,
			Ids:         nil,
			PartialName: environment.Name,
			Skip:        0,
			Take:        1,
		}
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve an account called \""+environment.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.environments) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c EnvironmentConverter) toHcl(environment octopus2.Environment, recursive bool, dependencies *ResourceDetailsCollection) error {
	resourceName := "environment_" + sanitizer.SanitizeName(environment.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = environment.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_environment." + resourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
			Type: hclsyntax.TokenComment,
			Bytes: []byte("# Import existing resources with the following commands:\n" +
				"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + environment.Name + "\") | .Id')\n" +
				"# terraform import octopusdeploy_environment." + resourceName + " ${RESOURCE_ID}\n"),
			SpacesBefore: 0,
		}})

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
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		// Add a data lookup to allow projects to quickly switch to using existing environments
		file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
			Type: hclsyntax.TokenComment,
			Bytes: []byte("# To use an existing environment, delete the resource above and use the following lookup instead:\n" +
				"# data.octopusdeploy_environments." + resourceName + ".environments[0].id\n"),
			SpacesBefore: 0,
		}})
		terraformDataResource := terraform.TerraformEnvironmentData{
			Type:        "octopusdeploy_environments",
			Name:        resourceName,
			Ids:         nil,
			PartialName: environment.Name,
			Skip:        0,
			Take:        1,
		}
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformDataResource, "data"))

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c EnvironmentConverter) getServiceNowChangeControlled(env octopus2.Environment) bool {
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

func (c EnvironmentConverter) getJiraServiceManagementExtensionSettings(env octopus2.Environment) bool {
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

func (c EnvironmentConverter) getJiraExtensionSettings(env octopus2.Environment) string {
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
