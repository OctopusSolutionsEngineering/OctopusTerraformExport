package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

const octopusdeployOfflinePackageDropDeploymentTargetDataType = "octopusdeploy_deployment_targets"
const octopusdeployOfflinePackageDropDeploymentTargetResourceType = "octopusdeploy_offline_package_drop_deployment_target"

type OfflineDropTargetConverter struct {
	Client                    client.OctopusClient
	MachinePolicyConverter    ConverterWithStatelessById
	EnvironmentConverter      ConverterAndLookupWithStatelessById
	ExcludeAllTargets         bool
	ExcludeTargets            args.ExcludeTargets
	ExcludeTargetsRegex       args.ExcludeTargets
	ExcludeTargetsExcept      args.ExcludeTargets
	DummySecretVariableValues bool
	DummySecretGenerator      DummySecretGenerator
	ExcludeTenantTags         args.ExcludeTenantTags
	ExcludeTenantTagSets      args.ExcludeTenantTagSets
	Excluder                  ExcludeByName
	TagSetConverter           ConvertToHclByResource[octopus.TagSet]
}

func (c OfflineDropTargetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) error {
	return c.allToHcl(false, dependencies)
}

func (c OfflineDropTargetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) error {
	return c.allToHcl(true, dependencies)
}

func (c OfflineDropTargetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	collection := octopus.GeneralCollection[octopus.OfflineDropResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	targets := lo.Filter(collection.Items, func(item octopus.OfflineDropResource, index int) bool {
		return c.isOfflineTarget(item)
	})

	for _, resource := range targets {
		zap.L().Info("Offline Drop Target: " + resource.Id)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c OfflineDropTargetConverter) isOfflineTarget(resource octopus.OfflineDropResource) bool {
	return resource.Endpoint.CommunicationStyle == "OfflineDrop"
}

func (c OfflineDropTargetConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c OfflineDropTargetConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c OfflineDropTargetConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
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

	if !c.isOfflineTarget(resource) {
		return nil
	}

	zap.L().Info("Offline Drop Target: " + resource.Id)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c OfflineDropTargetConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
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

	if !c.isOfflineTarget(resource) {
		return nil
	}

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if !c.isOfflineTarget(resource) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
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

func (c OfflineDropTargetConverter) toHcl(target octopus.OfflineDropResource, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(target.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if !c.isOfflineTarget(target) {
		return nil
	}

	if recursive {
		if stateless {
			if err := c.exportStatelessDependencies(target, dependencies); err != nil {
				return err
			}
		} else {
			if err := c.exportDependencies(target, dependencies); err != nil {
				return err
			}
		}
	}

	targetName := "target_" + sanitizer.SanitizeName(target.Name)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + targetName + ".tf"
	thisResource.Id = target.Id
	thisResource.Name = target.Name
	thisResource.ResourceType = c.GetResourceType()

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployOfflinePackageDropDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 " +
			"? data." + octopusdeployOfflinePackageDropDeploymentTargetDataType + "." + targetName + ".deployment_targets[0].id " +
			": " + octopusdeployOfflinePackageDropDeploymentTargetResourceType + "." + targetName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployOfflinePackageDropDeploymentTargetResourceType + "." + targetName + "}"
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

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, block, dependencies, recursive)
		if err != nil {
			return "", err
		}
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c OfflineDropTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c OfflineDropTargetConverter) lookupEnvironments(envs []string, dependencies *data.ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c OfflineDropTargetConverter) getMachinePolicy(machine string, dependencies *data.ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c OfflineDropTargetConverter) exportDependencies(target octopus.OfflineDropResource, dependencies *data.ResourceDetailsCollection) error {

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

func (c OfflineDropTargetConverter) exportStatelessDependencies(target octopus.OfflineDropResource, dependencies *data.ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclStatelessById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the environments
	for _, e := range target.EnvironmentIds {
		err = c.EnvironmentConverter.ToHclStatelessById(e, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}
