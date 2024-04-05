package converters

import (
	"fmt"
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

const octopusdeployProjectDeploymentTargetTriggerResourceType = "octopusdeploy_project_deployment_target_trigger"

type ProjectTriggerConverter struct {
	Client             client.OctopusClient
	LimitResourceCount int
	IncludeIds         bool
}

func (c ProjectTriggerConverter) ToHclByProjectIdAndName(projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.ProjectTrigger]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Project Trigger: " + resource.Id)
		err = c.toHcl(resource, false, false, projectId, projectName, dependencies)
		if err != nil {
			return err
		}
	}

	return nil
}

// We consider triggers to be the responsibility of a project. If the project exists, we don't create the trigger.
func (c ProjectTriggerConverter) buildData(resourceName string, name string) terraform.TerraformProjectData {
	return terraform.TerraformProjectData{
		Type:        octopusdeployProjectsDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c ProjectTriggerConverter) writeData(file *hclwrite.File, name string, resourceName string) {
	terraformResource := c.buildData(resourceName, name)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c ProjectTriggerConverter) toHcl(projectTrigger octopus2.ProjectTrigger, _ bool, stateless bool, projectId string, projectName string, dependencies *data.ResourceDetailsCollection) error {
	// Scheduled triggers with types like "OnceDailySchedule" are not supported
	if projectTrigger.Filter.FilterType != "MachineFilter" {
		zap.L().Error("Found an unsupported trigger type " + projectTrigger.Filter.FilterType)
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + projectTrigger.Id)
		return nil
	}

	projectTriggerName := "projecttrigger_" + sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(projectTrigger.Name)

	thisResource := data.ResourceDetails{}
	thisResource.Name = projectTrigger.Name
	thisResource.FileName = "space_population/" + projectTriggerName + ".tf"
	thisResource.Id = projectTrigger.Id
	thisResource.ResourceType = c.GetGroupResourceType(projectId)
	thisResource.Lookup = "${" + octopusdeployProjectDeploymentTargetTriggerResourceType + "." + projectTriggerName + ".id}"

	if stateless {
		// There is no way to look up an existing trigger. If the project exists, the lookup is an empty string. But
		// if the project exists, nothing will be created that needs to look up the trigger anyway.
		thisResource.Lookup = "${length(data." + octopusdeployProjectsDataType + "." + projectTriggerName + ".projects) != 0 " +
			"? null " +
			": " + octopusdeployProjectDeploymentTargetTriggerResourceType + "." + projectTriggerName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployProjectDeploymentTargetTriggerResourceType + "." + projectTriggerName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployProjectDeploymentTargetTriggerResourceType + "." + projectTriggerName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformProjectTrigger{
			Type:            octopusdeployProjectDeploymentTargetTriggerResourceType,
			Name:            projectTriggerName,
			Id:              strutil.InputPointerIfEnabled(c.IncludeIds, &projectTrigger.Id),
			ResourceName:    projectTrigger.Name,
			ProjectId:       dependencies.GetResource("Projects", projectTrigger.ProjectId),
			EventCategories: projectTrigger.Filter.EventCategories,
			EnvironmentIds:  projectTrigger.Filter.EnvironmentIds,
			EventGroups:     projectTrigger.Filter.EventGroups,
			Roles:           projectTrigger.Filter.Roles,
			ShouldRedeploy:  projectTrigger.Action.ShouldRedeployWhenMachineHasBeenDeployedTo,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			// when importing a stateless project, the trigger is only created if the project does not exist
			c.writeData(file, projectName, projectTriggerName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployProjectsDataType + "." + projectTriggerName + ".projects) != 0 ? 0 : 1}")
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

func (c ProjectTriggerConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/Triggers"
}

func (c ProjectTriggerConverter) GetResourceType() string {
	return "ProjectTriggers"
}
