package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

const octopusdeployOfflinePackageDropDeploymentTargetDataType = "octopusdeploy_deployment_targets"
const octopusdeployOfflinePackageDropDeploymentTargetResourceType = "octopusdeploy_offline_package_drop_deployment_target"

type OfflineDropTargetConverter struct {
	Client                    client.OctopusClient
	MachinePolicyConverter    ConverterById
	EnvironmentConverter      ConverterById
	ExcludeAllTargets         bool
	ExcludeTargets            args.ExcludeTargets
	ExcludeTargetsRegex       args.ExcludeTargets
	ExcludeTargetsExcept      args.ExcludeTargets
	DummySecretVariableValues bool
	DummySecretGenerator      DummySecretGenerator
	ExcludeTenantTags         args.ExcludeTenantTags
	ExcludeTenantTagSets      args.ExcludeTenantTagSets
	Excluder                  ExcludeByName
	TagSetConverter           TagSetConverter
}

func (c OfflineDropTargetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.OfflineDropResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Offline Drop Target: " + resource.Id)
		err = c.toHcl(resource, false, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c OfflineDropTargetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.OfflineDropResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Offline Drop Target: " + resource.Id)
	return c.toHcl(resource, true, false, dependencies)
}

func (c OfflineDropTargetConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.OfflineDropResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if resource.Endpoint.CommunicationStyle != "OfflineDrop" {
		return nil
	}

	thisResource := ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployOfflinePackageDropDeploymentTargetDataType + "." + resourceName + ".deployment_targets[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, resource)
		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c OfflineDropTargetConverter) buildData(resourceName string, resource octopus.OfflineDropResource) terraform.TerraformDeploymentTargetsData {
	return terraform.TerraformDeploymentTargetsData{
		Type:        octopusdeployOfflinePackageDropDeploymentTargetDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c OfflineDropTargetConverter) writeData(file *hclwrite.File, resource octopus.OfflineDropResource, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c OfflineDropTargetConverter) toHcl(target octopus.OfflineDropResource, recursive bool, stateless bool, dependencies *ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(target.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if target.Endpoint.CommunicationStyle == "OfflineDrop" {
		if recursive {
			err := c.exportDependencies(target, dependencies)

			if err != nil {
				return err
			}
		}

		targetName := "target_" + sanitizer.SanitizeName(target.Name)

		thisResource := ResourceDetails{}
		thisResource.FileName = "space_population/" + targetName + ".tf"
		thisResource.Id = target.Id
		thisResource.ResourceType = c.GetResourceType()

		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployOfflinePackageDropDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 " +
				"? data." + octopusdeployOfflinePackageDropDeploymentTargetDataType + "." + targetName + ".deployment_targets[0].id " +
				": " + octopusdeployOfflinePackageDropDeploymentTargetResourceType + "." + targetName + "[0].id}"
		} else {
			thisResource.Lookup = "${" + octopusdeployOfflinePackageDropDeploymentTargetResourceType + "." + targetName + ".id}"
		}

		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformOfflineDropDeploymentTarget{
				Type:                            octopusdeployOfflinePackageDropDeploymentTargetResourceType,
				Name:                            targetName,
				ApplicationsDirectory:           target.Endpoint.ApplicationsDirectory,
				WorkingDirectory:                target.Endpoint.OctopusWorkingDirectory,
				Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
				ResourceName:                    target.Name,
				Roles:                           target.Roles,
				HealthStatus:                    &target.HealthStatus,
				IsDisabled:                      &target.IsDisabled,
				MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId, dependencies),
				OperatingSystem:                 nil,
				ShellName:                       &target.ShellName,
				ShellVersion:                    &target.ShellVersion,
				SpaceId:                         nil,
				Status:                          nil,
				StatusSummary:                   nil,
				TenantTags:                      c.Excluder.FilteredTenantTags(target.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
				TenantedDeploymentParticipation: &target.TenantedDeploymentParticipation,
				Tenants:                         dependencies.GetResources("Tenants", target.TenantIds...),
				Thumbprint:                      nil,
				Uri:                             nil,
			}
			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, target, targetName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployOfflinePackageDropDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 ? 0 : 1}")
			}

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), target.Name, octopusdeployOfflinePackageDropDeploymentTargetResourceType, targetName))

			targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")
			err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, targetBlock, dependencies, recursive)
			if err != nil {
				return "", err
			}
			file.Body().AppendBlock(targetBlock)

			return string(file.Bytes()), nil
		}

		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c OfflineDropTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c OfflineDropTargetConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c OfflineDropTargetConverter) getMachinePolicy(machine string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c OfflineDropTargetConverter) exportDependencies(target octopus.OfflineDropResource, dependencies *ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the environments
	for _, e := range target.EnvironmentIds {
		err = c.EnvironmentConverter.ToHclById(e, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}
