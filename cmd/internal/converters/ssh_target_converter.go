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

const octopusdeploySshConnectionDeploymentTargetDataType = "octopusdeploy_deployment_targets"
const octopusdeploySshConnectionDeploymentTargetResourceType = "octopusdeploy_ssh_connection_deployment_target"

type SshTargetConverter struct {
	Client                 client.OctopusClient
	MachinePolicyConverter ConverterById
	AccountConverter       ConverterById
	EnvironmentConverter   ConverterById
	ExcludeAllTargets      bool
	ExcludeTargets         args.ExcludeTargets
	ExcludeTargetsRegex    args.ExcludeTargets
	ExcludeTargetsExcept   args.ExcludeTargets
	ExcludeTenantTags      args.ExcludeTenantTags
	ExcludeTenantTagSets   args.ExcludeTenantTagSets
	Excluder               ExcludeByName
	TagSetConverter        TagSetConverter
}

func (c SshTargetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.SshEndpointResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("SSH Target: " + resource.Id)
		err = c.toHcl(resource, false, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c SshTargetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.SshEndpointResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("SSH Target: " + resource.Id)
	return c.toHcl(resource, true, false, dependencies)
}

func (c SshTargetConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.SshEndpointResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if resource.Endpoint.CommunicationStyle != "Ssh" {
		return nil
	}

	thisResource := ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeploySshConnectionDeploymentTargetDataType + "." + resourceName + ".deployment_targets[0].id}"
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

func (c SshTargetConverter) buildData(resourceName string, resource octopus.SshEndpointResource) terraform.TerraformDeploymentTargetsData {
	return terraform.TerraformDeploymentTargetsData{
		Type:        octopusdeploySshConnectionDeploymentTargetDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c SshTargetConverter) writeData(file *hclwrite.File, resource octopus.SshEndpointResource, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c SshTargetConverter) toHcl(target octopus.SshEndpointResource, recursive bool, stateless bool, dependencies *ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(target.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if target.Endpoint.CommunicationStyle == "Ssh" {

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
			thisResource.Lookup = "${length(data." + octopusdeploySshConnectionDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 " +
				"? data." + octopusdeploySshConnectionDeploymentTargetDataType + "." + targetName + ".deployment_targets[0].id " +
				": " + octopusdeploySshConnectionDeploymentTargetResourceType + "." + targetName + "[0].id}"
		} else {
			thisResource.Lookup = "${" + octopusdeploySshConnectionDeploymentTargetResourceType + "." + targetName + ".id}"
		}

		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformSshConnectionDeploymentTarget{
				Type:               octopusdeploySshConnectionDeploymentTargetResourceType,
				Name:               targetName,
				AccountId:          c.getAccount(target.Endpoint.AccountId, dependencies),
				Environments:       c.lookupEnvironments(target.EnvironmentIds, dependencies),
				Fingerprint:        target.Endpoint.Fingerprint,
				Host:               target.Endpoint.Host,
				ResourceName:       target.Name,
				Roles:              target.Roles,
				DotNetCorePlatform: &target.Endpoint.DotNetCorePlatform,
				MachinePolicyId:    c.getMachinePolicy(target.MachinePolicyId, dependencies),
				TenantTags:         c.Excluder.FilteredTenantTags(target.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
			}
			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, target, targetName)
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeploySshConnectionDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 ? 0 : 1}")
			}

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), target.Name, octopusdeploySshConnectionDeploymentTargetResourceType, targetName))

			block := gohcl.EncodeAsBlock(terraformResource, "resource")
			err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, block, dependencies, recursive)
			if err != nil {
				return "", err
			}
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}

		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c SshTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c SshTargetConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c SshTargetConverter) getMachinePolicy(machine string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c SshTargetConverter) getAccount(account string, dependencies *ResourceDetailsCollection) string {
	accountLookup := dependencies.GetResource("Accounts", account)
	if accountLookup == "" {
		return ""
	}

	return accountLookup
}

func (c SshTargetConverter) exportDependencies(target octopus.SshEndpointResource, dependencies *ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the accounts
	err = c.AccountConverter.ToHclById(target.Endpoint.AccountId, dependencies)

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
