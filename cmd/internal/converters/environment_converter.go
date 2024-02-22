package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

const octopusdeployEnvironmentsDataType = "octopusdeploy_environments"
const octopusdeployEnvironmentsResourceType = "octopusdeploy_environment"

type EnvironmentConverter struct {
	Client client.OctopusClient
}

func (c EnvironmentConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) error {
	return c.allToHcl(false, dependencies)
}

func (c EnvironmentConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) error {
	return c.allToHcl(true, dependencies)
}

func (c EnvironmentConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.Environment]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Environment: " + resource.Id)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c EnvironmentConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.Environment{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Environment: " + resource.Id)
	return c.toHcl(resource, true, false, dependencies)
}

func (c EnvironmentConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
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

	thisResource := data.ResourceDetails{}

	resourceName := "environment_" + sanitizer.SanitizeName(environment.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = environment.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployEnvironmentsDataType + "." + resourceName + ".environments[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, environment)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve an environment called \""+environment.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.environments) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c EnvironmentConverter) buildData(resourceName string, resource octopus2.Environment) terraform.TerraformEnvironmentData {
	return terraform.TerraformEnvironmentData{
		Type:        octopusdeployEnvironmentsDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c EnvironmentConverter) writeData(file *hclwrite.File, resource octopus2.Environment, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c EnvironmentConverter) getLookup(stateless bool, resourceName string) string {
	if stateless {
		return "${length(data." + octopusdeployEnvironmentsDataType + "." + resourceName + ".environments) != 0 " +
			"? data." + octopusdeployEnvironmentsDataType + "." + resourceName + ".environments[0].id " +
			": " + octopusdeployEnvironmentsResourceType + "." + resourceName + "[0].id}"
	}
	return "${" + octopusdeployEnvironmentsResourceType + "." + resourceName + ".id}"

}

func (c EnvironmentConverter) getDependency(stateless bool, resourceName string) string {
	if stateless {
		return "${" + octopusdeployEnvironmentsResourceType + "." + resourceName + "}"
	}

	return ""
}

func (c EnvironmentConverter) getCount(stateless bool, resourceName string) *string {
	if stateless {
		return strutil.StrPointer("${length(data." + octopusdeployEnvironmentsDataType + "." + resourceName + ".environments) != 0 ? 0 : 1}")
	}

	return nil
}

func (c EnvironmentConverter) toHcl(environment octopus2.Environment, _ bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	resourceName := "environment_" + sanitizer.SanitizeName(environment.Name)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = environment.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = c.getLookup(stateless, resourceName)
	thisResource.Dependency = c.getDependency(stateless, resourceName)

	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), environment.Name, octopusdeployEnvironmentsResourceType, resourceName))

		terraformResource := terraform.TerraformEnvironment{
			Type:                       octopusdeployEnvironmentsResourceType,
			Name:                       resourceName,
			Count:                      c.getCount(stateless, resourceName),
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

		if stateless {
			c.writeData(file, environment, resourceName)
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

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
