package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type DeploymentProcessConverter struct {
	Client client.OctopusClient
}

func (c DeploymentProcessConverter) ToHclById(id string, parentName string) (map[string]string, error) {
	resource := model.DeploymentProcess{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return nil, err
	}

	resourceName := "octopus_deployment_process_" + util.SanitizeName(resource.Id)

	terraformResource := model.TerraformDeploymentProcess{
		Type:      "octopusdeploy_deployment_process",
		Name:      resourceName,
		ProjectId: "octopusdeploy_project." + parentName + ".id",
		Step:      make([]model.TerraformStep, len(resource.Steps)),
	}

	for i, s := range resource.Steps {
		terraformResource.Step[i] = model.TerraformStep{
			Name:               s.Name,
			PackageRequirement: s.PackageRequirement,
			Properties:         s.Properties,
			Condition:          s.Condition,
			StartTrigger:       s.StartTrigger,
			Action:             make([]model.TerraformAction, len(s.Actions)),
		}

		for j, a := range s.Actions {
			terraformResource.Step[i].Action[j] = model.TerraformAction{
				Name:                          a.Name,
				ActionType:                    a.ActionType,
				Notes:                         a.Notes,
				IsDisabled:                    a.IsDisabled,
				CanBeUsedForProjectVersioning: a.CanBeUsedForProjectVersioning,
				IsRequired:                    a.IsRequired,
				WorkerPoolId:                  a.WorkerPoolId,
				Container: model.TerraformContainer{
					FeedId: a.Container.FeedId,
					Image:  a.Container.Image,
				},
				WorkerPoolVariable:   a.WorkerPoolVariable,
				Environments:         a.Environments,
				ExcludedEnvironments: a.ExcludedEnvironments,
				Channels:             a.Channels,
				TenantTags:           a.TenantTags,
				Package:              make([]model.TerraformPackage, len(a.Packages)),
				Condition:            a.Condition,
				Properties:           a.Properties,
			}

			for k, p := range a.Packages {
				terraformResource.Step[i].Action[j].Package[k] = model.TerraformPackage{
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
		internal.PopulateSpaceDir + "/" + resourceName + ".tf": string(file.Bytes()),
	}, nil
}

func (c DeploymentProcessConverter) GetResourceType() string {
	return "DeploymentProcesses"
}
