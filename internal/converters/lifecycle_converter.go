package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type LifecycleConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
}

func (c LifecycleConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.Lifecycle]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	lifecycleMap := map[string]string{}

	for _, lifecycle := range collection.Items {
		resourceName := "lifecycle_" + util.SanitizeNamePointer(&lifecycle.Name)

		// Assume the default lifecycle already exists
		if lifecycle.Name == "Default Lifecycle" {
			data := terraform.TerraformLifecycleData{
				Type:        "octopusdeploy_lifecycles",
				Name:        resourceName,
				Ids:         nil,
				PartialName: lifecycle.Name,
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(data, "data"))

			results["space_population/"+resourceName+".tf"] = string(file.Bytes())
			lifecycleMap[lifecycle.Id] = "${data.octopusdeploy_lifecycles." + resourceName + ".lifecycles[0].id}"
		} else {
			terraformResource := terraform.TerraformLifecycle{
				Type:         "octopusdeploy_lifecycle",
				Name:         resourceName,
				ResourceName: lifecycle.Name,
				Description:  lifecycle.Description,
				Phase:        c.convertPhases(lifecycle.Phases),
				ReleaseRetentionPolicy: terraform.TerraformPolicy{
					QuantityToKeep:    lifecycle.ReleaseRetentionPolicy.QuantityToKeep,
					ShouldKeepForever: lifecycle.ReleaseRetentionPolicy.ShouldKeepForever,
					Unit:              lifecycle.ReleaseRetentionPolicy.Unit,
				},
				TentacleRetentionPolicy: terraform.TerraformPolicy{
					QuantityToKeep:    lifecycle.TentacleRetentionPolicy.QuantityToKeep,
					ShouldKeepForever: lifecycle.TentacleRetentionPolicy.ShouldKeepForever,
					Unit:              lifecycle.TentacleRetentionPolicy.Unit,
				},
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			results["space_population/"+resourceName+".tf"] = string(file.Bytes())
			lifecycleMap[lifecycle.Id] = "${octopusdeploy_lifecycle." + resourceName + ".id}"
		}
	}

	return results, lifecycleMap, nil
}

func (c LifecycleConverter) ToHclById(id string) (map[string]string, error) {
	lifecycle := octopus.Lifecycle{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &lifecycle)

	if err != nil {
		return nil, err
	}

	resourceName := "lifecycle_" + util.SanitizeNamePointer(&lifecycle.Name)
	terraformResource := terraform.TerraformLifecycle{
		Type:         "octopusdeploy_lifecycle",
		Name:         resourceName,
		ResourceName: lifecycle.Name,
		Description:  lifecycle.Description,
		Phase:        c.convertPhases(lifecycle.Phases),
		ReleaseRetentionPolicy: terraform.TerraformPolicy{
			QuantityToKeep:    lifecycle.ReleaseRetentionPolicy.QuantityToKeep,
			ShouldKeepForever: lifecycle.ReleaseRetentionPolicy.ShouldKeepForever,
			Unit:              lifecycle.ReleaseRetentionPolicy.Unit,
		},
		TentacleRetentionPolicy: terraform.TerraformPolicy{
			QuantityToKeep:    lifecycle.TentacleRetentionPolicy.QuantityToKeep,
			ShouldKeepForever: lifecycle.TentacleRetentionPolicy.ShouldKeepForever,
			Unit:              lifecycle.TentacleRetentionPolicy.Unit,
		},
	}
	file := hclwrite.NewEmptyFile()
	file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

	return map[string]string{
		resourceName + ".tf": string(file.Bytes()),
	}, nil
}

func (c LifecycleConverter) ToHclByName(name string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (c LifecycleConverter) GetResourceType() string {
	return "Lifecycles"
}

func (c LifecycleConverter) convertPhases(phases []octopus.Phase) []terraform.TerraformPhase {
	terraformPhases := make([]terraform.TerraformPhase, 0)
	for _, v := range phases {
		terraformPhases = append(terraformPhases, terraform.TerraformPhase{
			AutomaticDeploymentTargets:         v.AutomaticDeploymentTargets,
			OptionalDeploymentTargets:          v.OptionalDeploymentTargets,
			Name:                               v.Name,
			IsOptionalPhase:                    v.IsOptionalPhase,
			MinimumEnvironmentsBeforePromotion: v.MinimumEnvironmentsBeforePromotion,
			ReleaseRetentionPolicy: terraform.TerraformPolicy{
				QuantityToKeep:    v.ReleaseRetentionPolicy.QuantityToKeep,
				ShouldKeepForever: v.ReleaseRetentionPolicy.ShouldKeepForever,
				Unit:              v.ReleaseRetentionPolicy.Unit,
			},
			TentacleRetentionPolicy: terraform.TerraformPolicy{
				QuantityToKeep:    v.TentacleRetentionPolicy.QuantityToKeep,
				ShouldKeepForever: v.TentacleRetentionPolicy.ShouldKeepForever,
				Unit:              v.TentacleRetentionPolicy.Unit,
			},
		})
	}
	return terraformPhases
}
