package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type DeploymentProcessConverter struct {
	Client client.OctopusClient
}

func (c DeploymentProcessConverter) ToHclById(id string, parentName string) (map[string]string, error) {
	resource := octopus.DeploymentProcess{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, err
	}

	resourceName := util.SanitizeName(resource.Id)

	terraformResource := terraform.TerraformDeploymentProcess{
		Type:      "octopusdeploy_deployment_process",
		Name:      resourceName,
		ProjectId: "octopusdeploy_project." + parentName + ".id",
		Step:      make([]terraform.TerraformStep, len(resource.Steps)),
	}

	for i, s := range resource.Steps {
		terraformResource.Step[i] = terraform.TerraformStep{
			Name:               s.Name,
			PackageRequirement: s.PackageRequirement,
			Properties:         s.Properties,
			Condition:          s.Condition,
			StartTrigger:       s.StartTrigger,
			Action:             make([]terraform.TerraformAction, len(s.Actions)),
		}

		for j, a := range s.Actions {
			terraformResource.Step[i].Action[j] = terraform.TerraformAction{
				Name:                          a.Name,
				ActionType:                    a.ActionType,
				Notes:                         a.Notes,
				IsDisabled:                    a.IsDisabled,
				CanBeUsedForProjectVersioning: a.CanBeUsedForProjectVersioning,
				IsRequired:                    a.IsRequired,
				WorkerPoolId:                  a.WorkerPoolId,
				Container:                     c.convertContainer(a.Container),
				WorkerPoolVariable:            a.WorkerPoolVariable,
				Environments:                  a.Environments,
				ExcludedEnvironments:          a.ExcludedEnvironments,
				Channels:                      a.Channels,
				TenantTags:                    a.TenantTags,
				Package:                       make([]terraform.TerraformPackage, len(a.Packages)),
				Condition:                     a.Condition,
				Properties:                    a.Properties,
			}

			for k, p := range a.Packages {
				terraformResource.Step[i].Action[j].Package[k] = terraform.TerraformPackage{
					Name:                    p.Name,
					PackageID:               p.PackageId,
					AcquisitionLocation:     p.AcquisitionLocation,
					ExtractDuringDeployment: p.ExtractDuringDeployment,
					FeedId:                  p.FeedId,
					Id:                      p.Id,
					Properties:              p.Properties,
				}
			}
		}
	}

	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

	return map[string]string{
		resourceName + ".tf": string(file.Bytes()),
	}, nil
}

func (c DeploymentProcessConverter) GetResourceType() string {
	return "DeploymentProcesses"
}

func (c DeploymentProcessConverter) convertContainer(container octopus.Container) *terraform.TerraformContainer {
	if container.Image != nil || container.FeedId != nil {
		return &terraform.TerraformContainer{
			FeedId: container.FeedId,
			Image:  container.Image,
		}
	}

	return nil
}
