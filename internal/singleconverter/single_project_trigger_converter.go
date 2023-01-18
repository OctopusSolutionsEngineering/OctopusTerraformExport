package singleconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type SingleProjectTriggerConverter struct {
	Client client.OctopusClient
}

func (c SingleProjectTriggerConverter) ToHcl(projectId string, projectName string, dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.ProjectTrigger]{}
	err := c.Client.GetAllResources(c.GetResourceType(projectId), &collection)

	if err != nil {
		return err
	}

	for _, projectTrigger := range collection.Items {
		projectTriggerName := "projecttrigger_" + util.SanitizeName(projectName) + "_" + util.SanitizeName(projectTrigger.Name)

		thisResource := ResourceDetails{}
		thisResource.FileName = "space_population/" + projectTriggerName + ".tf"
		thisResource.Id = projectTrigger.Id
		thisResource.ResourceType = c.GetResourceType(projectId)
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
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}
	}

	return nil
}

func (c SingleProjectTriggerConverter) GetResourceType(projectId string) string {
	return "Projects/" + projectId + "/Triggers"
}
