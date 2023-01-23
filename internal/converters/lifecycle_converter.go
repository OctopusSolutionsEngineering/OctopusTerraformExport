package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/sanitizer"
)

type LifecycleConverter struct {
	Client               client.OctopusClient
	EnvironmentConverter ConverterById
}

func (c LifecycleConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Lifecycle]{}
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

func (c LifecycleConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	// Channels can have empty strings for the lifecycle ID
	if id == "" {
		return nil
	}

	lifecycle := octopus.Lifecycle{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &lifecycle)

	if err != nil {
		return err
	}

	return c.toHcl(lifecycle, true, dependencies)

}

func (c LifecycleConverter) toHcl(lifecycle octopus.Lifecycle, recursive bool, dependencies *ResourceDetailsCollection) error {

	if recursive {
		// The environments are a dependency that we need to lookup
		for _, phase := range lifecycle.Phases {
			for _, auto := range phase.AutomaticDeploymentTargets {
				err := c.EnvironmentConverter.ToHclById(auto, dependencies)

				if err != nil {
					return err
				}
			}
			for _, optional := range phase.OptionalDeploymentTargets {
				err := c.EnvironmentConverter.ToHclById(optional, dependencies)

				if err != nil {
					return err
				}
			}
		}
	}

	resourceName := "lifecycle_" + sanitizer.SanitizeName(lifecycle.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = lifecycle.Id
	thisResource.ResourceType = c.GetResourceType()
	if lifecycle.Name == "Default Lifecycle" {
		thisResource.Lookup = "${data.octopusdeploy_lifecycles." + resourceName + ".lifecycles[0].id}"
	} else {
		thisResource.Lookup = "${octopusdeploy_lifecycle." + resourceName + ".id}"
	}
	thisResource.ToHcl = func() (string, error) {
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

			return string(file.Bytes()), nil
		} else {
			terraformResource := terraform.TerraformLifecycle{
				Type:         "octopusdeploy_lifecycle",
				Name:         resourceName,
				ResourceName: lifecycle.Name,
				Description:  lifecycle.Description,
				Phase:        c.convertPhases(lifecycle.Phases, dependencies),
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

			return string(file.Bytes()), nil
		}
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c LifecycleConverter) GetResourceType() string {
	return "Lifecycles"
}

func (c LifecycleConverter) convertPhases(phases []octopus.Phase, dependencies *ResourceDetailsCollection) []terraform.TerraformPhase {
	terraformPhases := make([]terraform.TerraformPhase, 0)
	for _, v := range phases {
		terraformPhases = append(terraformPhases, terraform.TerraformPhase{
			AutomaticDeploymentTargets:         c.convertTargets(v.AutomaticDeploymentTargets, dependencies),
			OptionalDeploymentTargets:          c.convertTargets(v.OptionalDeploymentTargets, dependencies),
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

func (c LifecycleConverter) convertTargets(environments []string, dependencies *ResourceDetailsCollection) []string {
	converted := make([]string, len(environments))

	for i, v := range environments {
		converted[i] = dependencies.GetResource("Environments", v)
	}

	return converted
}
