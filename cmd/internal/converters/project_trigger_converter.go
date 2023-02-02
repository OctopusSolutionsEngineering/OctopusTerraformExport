package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
)

type ProjectTriggerConverter struct {
	Client client.OctopusClient
}

func (c ProjectTriggerConverter) ToHclByProjectIdAndName(projectId string, projectName string, dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.ProjectTrigger]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return err
	}

	for _, projectTrigger := range collection.Items {
		err = c.toHcl(projectTrigger, false, projectId, projectName, dependencies)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c ProjectTriggerConverter) toHcl(projectTrigger octopus2.ProjectTrigger, recursive bool, projectId string, projectName string, dependencies *ResourceDetailsCollection) error {
	// Scheduled triggers with types like "OnceDailySchedule" are not supported
	if projectTrigger.Filter.FilterType != "MachineFilter" {
		fmt.Println("Found an unsupported trigger type " + projectTrigger.Filter.FilterType)
		return nil
	}

	projectTriggerName := "projecttrigger_" + sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(projectTrigger.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + projectTriggerName + ".tf"
	thisResource.Id = projectTrigger.Id
	thisResource.ResourceType = c.GetGroupResourceType(projectId)
	thisResource.Lookup = "${octopusdeploy_project_deployment_target_trigger." + projectTriggerName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformProjectTrigger{
			Type:            "octopusdeploy_project_deployment_target_trigger",
			Name:            projectTriggerName,
			ResourceName:    projectTrigger.Name,
			ProjectId:       dependencies.GetResource("Projects", projectTrigger.ProjectId),
			EventCategories: projectTrigger.Filter.EventCategories,
			EnvironmentIds:  projectTrigger.Filter.EnvironmentIds,
			EventGroups:     projectTrigger.Filter.EventGroups,
			Roles:           projectTrigger.Filter.Roles,
			ShouldRedeploy:  projectTrigger.Action.ShouldRedeployWhenMachineHasBeenDeployedTo,
			Id:              nil,
		}
		file := hclwrite.NewEmptyFile()

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
			Type: hclsyntax.TokenComment,
			Bytes: []byte("# Import existing resources with the following commands:\n" +
				"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.name=\"" + projectTrigger.Name + "\") | .Id')\n" +
				"# terraform import octopusdeploy_project_deployment_target_trigger." + projectTriggerName + " ${RESOURCE_ID}\n"),
			SpacesBefore: 0,
		}})

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c ProjectTriggerConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/Triggers"
}

func (c ProjectTriggerConverter) GetResourceType() string {
	return "ProjectTriggers"
}
