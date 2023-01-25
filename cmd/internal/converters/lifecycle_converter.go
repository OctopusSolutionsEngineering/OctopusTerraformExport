package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
)

type LifecycleConverter struct {
	Client               client.OctopusClient
	EnvironmentConverter ConverterById
}

func (c LifecycleConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.Lifecycle]{}
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

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	lifecycle := octopus2.Lifecycle{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &lifecycle)

	if err != nil {
		return err
	}

	return c.toHcl(lifecycle, true, dependencies)

}

func (c LifecycleConverter) toHcl(lifecycle octopus2.Lifecycle, recursive bool, dependencies *ResourceDetailsCollection) error {

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
			data := terraform2.TerraformLifecycleData{
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
			terraformResource := terraform2.TerraformLifecycle{
				Type:                    "octopusdeploy_lifecycle",
				Name:                    resourceName,
				ResourceName:            lifecycle.Name,
				Description:             lifecycle.Description,
				Phase:                   c.convertPhases(lifecycle.Phases, dependencies),
				ReleaseRetentionPolicy:  c.convertPolicy(lifecycle.ReleaseRetentionPolicy),
				TentacleRetentionPolicy: c.convertPolicy(lifecycle.TentacleRetentionPolicy),
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

func (c LifecycleConverter) convertPolicy(policy *octopus2.Policy) *terraform2.TerraformPolicy {
	if policy == nil {
		return nil
	}

	return &terraform2.TerraformPolicy{
		QuantityToKeep:    policy.QuantityToKeep,
		ShouldKeepForever: policy.ShouldKeepForever,
		Unit:              policy.Unit,
	}
}

func (c LifecycleConverter) convertPhases(phases []octopus2.Phase, dependencies *ResourceDetailsCollection) []terraform2.TerraformPhase {
	terraformPhases := make([]terraform2.TerraformPhase, 0)
	for _, v := range phases {
		terraformPhases = append(terraformPhases, terraform2.TerraformPhase{
			AutomaticDeploymentTargets:         c.convertTargets(v.AutomaticDeploymentTargets, dependencies),
			OptionalDeploymentTargets:          c.convertTargets(v.OptionalDeploymentTargets, dependencies),
			Name:                               v.Name,
			IsOptionalPhase:                    v.IsOptionalPhase,
			MinimumEnvironmentsBeforePromotion: v.MinimumEnvironmentsBeforePromotion,
			ReleaseRetentionPolicy:             c.convertPolicy(v.ReleaseRetentionPolicy),
			TentacleRetentionPolicy:            c.convertPolicy(v.TentacleRetentionPolicy),
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
