package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	octopus2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const octopusdeployLifecyclesDataType = "octopusdeploy_lifecycles"
const octopusdeployLifecycleResourceType = "octopusdeploy_lifecycle"

type LifecycleConverter struct {
	Client               client.OctopusClient
	EnvironmentConverter ConverterAndLookupWithStatelessById
	ErrGroup             *errgroup.Group
}

func (c LifecycleConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c LifecycleConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c LifecycleConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.Lifecycle]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Lifecycle: " + resource.Id)
		err = c.toHcl(resource, false, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c LifecycleConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c LifecycleConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c LifecycleConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Channels can have empty strings for the lifecycle ID
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.Lifecycle{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Lifecycle: " + resource.Id)
	return c.toHcl(resource, true, false, stateless, dependencies)

}

func (c LifecycleConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
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

	return c.toHcl(lifecycle, false, true, false, dependencies)

}

func (c LifecycleConverter) buildData(resourceName string, resource octopus2.Lifecycle) terraform2.TerraformLifecycleData {
	return terraform2.TerraformLifecycleData{
		Type:        octopusdeployLifecyclesDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c LifecycleConverter) writeData(file *hclwrite.File, resource octopus2.Lifecycle, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c LifecycleConverter) toHcl(lifecycle octopus2.Lifecycle, recursive bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {

	if recursive {
		// The environments are a dependency that we need to lookup
		for _, phase := range lifecycle.Phases {
			for _, auto := range phase.AutomaticDeploymentTargets {
				if stateless {
					err := c.EnvironmentConverter.ToHclStatelessById(auto, dependencies)
					if err != nil {
						return err
					}
				} else {
					err := c.EnvironmentConverter.ToHclById(auto, dependencies)
					if err != nil {
						return err
					}
				}

			}
			for _, optional := range phase.OptionalDeploymentTargets {
				if stateless {
					err := c.EnvironmentConverter.ToHclStatelessById(optional, dependencies)

					if err != nil {
						return err
					}
				} else {
					err := c.EnvironmentConverter.ToHclById(optional, dependencies)

					if err != nil {
						return err
					}
				}

			}
		}
	}

	forceLookup := lookup || lifecycle.Name == "Default Lifecycle"

	resourceName := "lifecycle_" + sanitizer.SanitizeName(lifecycle.Name)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = lifecycle.Id
	thisResource.Name = lifecycle.Name
	thisResource.ResourceType = c.GetResourceType()
	if forceLookup {
		thisResource.Lookup = "${data." + octopusdeployLifecyclesDataType + "." + resourceName + ".lifecycles[0].id}"

		thisResource.ToHcl = func() (string, error) {
			data := c.buildData(resourceName, lifecycle)
			file := hclwrite.NewEmptyFile()
			block := gohcl.EncodeAsBlock(data, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a lifecycle called \""+lifecycle.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.lifecycles) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	} else {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployLifecyclesDataType + "." + resourceName + ".lifecycles) != 0 " +
				"? data." + octopusdeployLifecyclesDataType + "." + resourceName + ".lifecycles[0].id " +
				": " + octopusdeployLifecycleResourceType + "." + resourceName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployLifecycleResourceType + "." + resourceName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployLifecycleResourceType + "." + resourceName + ".id}"
		}

		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform2.TerraformLifecycle{
				Type:                    octopusdeployLifecycleResourceType,
				Name:                    resourceName,
				ResourceName:            lifecycle.Name,
				Description:             lifecycle.Description,
				Phase:                   c.convertPhases(lifecycle.Phases, dependencies),
				ReleaseRetentionPolicy:  c.convertPolicy(lifecycle.ReleaseRetentionPolicy),
				TentacleRetentionPolicy: c.convertPolicy(lifecycle.TentacleRetentionPolicy),
			}
			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, lifecycle, resourceName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployLifecyclesDataType + "." + resourceName + ".lifecycles) != 0 ? 0 : 1}")
			}

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			if stateless {
				hcl.WriteLifecyclePreventDestroyAttribute(block)
			}

			file.Body().AppendBlock(block)

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

func (c LifecycleConverter) convertPhases(phases []octopus2.Phase, dependencies *data.ResourceDetailsCollection) []terraform2.TerraformPhase {
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

func (c LifecycleConverter) convertTargets(environments []string, dependencies *data.ResourceDetailsCollection) []string {
	converted := make([]string, len(environments))

	for i, v := range environments {
		converted[i] = dependencies.GetResource("Environments", v)
	}

	return converted
}
