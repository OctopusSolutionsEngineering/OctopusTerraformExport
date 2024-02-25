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

const octopusdeployListeningTentacleDeploymentTargetDataType = "octopusdeploy_deployment_targets"
const octopusdeployListeningTentacleDeploymentTargetResourceType = "octopusdeploy_listening_tentacle_deployment_target"

type ListeningTargetConverter struct {
	Client                 client.OctopusClient
	MachinePolicyConverter ConverterWithStatelessById
	EnvironmentConverter   ConverterAndLookupWithStatelessById
	ExcludeAllTargets      bool
	ExcludeTargets         args.ExcludeTargets
	ExcludeTargetsRegex    args.ExcludeTargets
	ExcludeTargetsExcept   args.ExcludeTargets
	ExcludeTenantTags      args.ExcludeTenantTags
	ExcludeTenantTagSets   args.ExcludeTenantTagSets
	Excluder               ExcludeByName
	TagSetConverter        ConvertToHclByResource[octopus.TagSet]
}

func (c ListeningTargetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) error {
	return c.allToHcl(false, dependencies)
}

func (c ListeningTargetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) error {
	return c.allToHcl(true, dependencies)
}

func (c ListeningTargetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	collection := octopus.GeneralCollection[octopus.ListeningEndpointResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	targets := lo.Filter(collection.Items, func(item octopus.ListeningEndpointResource, index int) bool {
		return c.isListeningTarget(item)
	})

	for _, resource := range targets {
		zap.L().Info("Listening Target: " + resource.Id)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c ListeningTargetConverter) isListeningTarget(resource octopus.ListeningEndpointResource) bool {
	return resource.Endpoint.CommunicationStyle == "TentaclePassive"
}

func (c ListeningTargetConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c ListeningTargetConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c ListeningTargetConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.ListeningEndpointResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if !c.isListeningTarget(resource) {
		return nil
	}

	zap.L().Info("Listening Target: " + resource.Id)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c ListeningTargetConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.ListeningEndpointResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if !c.isListeningTarget(resource) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployListeningTentacleDeploymentTargetDataType + "." + resourceName + ".deployment_targets[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, resource)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a deployment target called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.deployment_targets) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c ListeningTargetConverter) buildData(resourceName string, resource octopus.ListeningEndpointResource) terraform.TerraformDeploymentTargetsData {
	return terraform.TerraformDeploymentTargetsData{
		Type:        octopusdeployListeningTentacleDeploymentTargetDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c ListeningTargetConverter) writeData(file *hclwrite.File, resource octopus.ListeningEndpointResource, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c ListeningTargetConverter) toHcl(target octopus.ListeningEndpointResource, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(target.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if !c.isListeningTarget(target) {
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
		thisResource.Lookup = "${length(data." + octopusdeployListeningTentacleDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 " +
			"? data." + octopusdeployListeningTentacleDeploymentTargetDataType + "." + targetName + ".deployment_targets[0].id " +
			": " + octopusdeployListeningTentacleDeploymentTargetResourceType + "." + targetName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployListeningTentacleDeploymentTargetResourceType + "." + targetName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployListeningTentacleDeploymentTargetResourceType + "." + targetName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformListeningTentacleDeploymentTarget{
			Type:                            octopusdeployListeningTentacleDeploymentTargetResourceType,
			Name:                            targetName,
			Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
			ResourceName:                    target.Name,
			Roles:                           target.Roles,
			TentacleUrl:                     target.Uri,
			Thumbprint:                      target.Thumbprint,
			CertificateSignatureAlgorithm:   nil,
			HealthStatus:                    nil,
			IsDisabled:                      &target.IsDisabled,
			IsInProcess:                     &target.IsInProcess,
			MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId, dependencies),
			OperatingSystem:                 nil,
			ProxyId:                         nil,
			ShellName:                       &target.ShellName,
			ShellVersion:                    &target.ShellVersion,
			SpaceId:                         nil,
			Status:                          nil,
			StatusSummary:                   nil,
			TenantTags:                      c.Excluder.FilteredTenantTags(target.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			TenantedDeploymentParticipation: &target.TenantedDeploymentParticipation,
			Tenants:                         dependencies.GetResources("Tenants", target.TenantIds...),
			TentacleVersionDetails:          terraform.TerraformTentacleVersionDetails{},
			Uri:                             nil,
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, target, targetName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployListeningTentacleDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 ? 0 : 1}")
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), target.Name, octopusdeployListeningTentacleDeploymentTargetResourceType, targetName))

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

func (c ListeningTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c ListeningTargetConverter) lookupEnvironments(envs []string, dependencies *data.ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c ListeningTargetConverter) getMachinePolicy(machine string, dependencies *data.ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c ListeningTargetConverter) exportDependencies(target octopus.ListeningEndpointResource, dependencies *data.ResourceDetailsCollection) error {

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

func (c ListeningTargetConverter) exportStatelessDependencies(target octopus.ListeningEndpointResource, dependencies *data.ResourceDetailsCollection) error {

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
