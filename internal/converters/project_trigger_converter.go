package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type ProjectTriggerConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	ProjectId         string
	ProjectLookup     string
	ProjectName       string
}

func (c ProjectTriggerConverter) ToHcl() (map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.ProjectTrigger]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, err
	}

	results := map[string]string{}

	for _, projectTrigger := range collection.Items {
		projectTriggerName := "projecttrigger_" + util.SanitizeName(c.ProjectName) + "_" + util.SanitizeName(projectTrigger.Name)

		terraformResource := terraform.TerraformProjectTrigger{
			Type:            "octopusdeploy_project_deployment_target_trigger",
			Name:            projectTriggerName,
			ResourceName:    projectTrigger.Name,
			ProjectId:       c.ProjectLookup,
			EventCategories: projectTrigger.Filter.EventCategories,
			EnvironmentIds:  projectTrigger.Filter.EnvironmentIds,
			EventGroups:     projectTrigger.Filter.EventGroups,
			Roles:           projectTrigger.Filter.Roles,
			ShouldRedeploy:  projectTrigger.Action.ShouldRedeployWhenMachineHasBeenDeployedTo,
			Id:              nil,
		}
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		results["space_population/projecttrigger_"+projectTriggerName+".tf"] = string(file.Bytes())
	}

	return results, nil
}

func (c ProjectTriggerConverter) GetResourceType() string {
	return "Projects/" + c.ProjectId + "/Triggers"
}
