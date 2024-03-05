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
	"golang.org/x/sync/errgroup"
)

const azureCloudServiceDeploymentDataType = "octopusdeploy_deployment_targets"
const azureCloudServiceDeploymentResourceType = "octopusdeploy_azure_cloud_service_deployment_target"

type AzureCloudServiceTargetConverter struct {
	Client                 client.OctopusClient
	MachinePolicyConverter ConverterWithStatelessById
	AccountConverter       ConverterAndLookupWithStatelessById
	EnvironmentConverter   ConverterAndLookupWithStatelessById
	ExcludeAllTargets      bool
	ExcludeTargets         args.ExcludeTargets
	ExcludeTargetsRegex    args.ExcludeTargets
	ExcludeTargetsExcept   args.ExcludeTargets
	ExcludeTenantTags      args.ExcludeTenantTags
	ExcludeTenantTagSets   args.ExcludeTenantTagSets
	Excluder               ExcludeByName
	TagSetConverter        ConvertToHclByResource[octopus.TagSet]
	ErrGroup               *errgroup.Group
}

func (c AzureCloudServiceTargetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c AzureCloudServiceTargetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c AzureCloudServiceTargetConverter) isAzureCloudService(resource octopus.AzureCloudServiceResource) bool {
	return resource.Endpoint.CommunicationStyle == "AzureCloudService"
}

func (c AzureCloudServiceTargetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	collection := octopus.GeneralCollection[octopus.AzureCloudServiceResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	targets := lo.Filter(collection.Items, func(item octopus.AzureCloudServiceResource, index int) bool {
		return c.isAzureCloudService(item)
	})

	for _, resource := range targets {
		zap.L().Info("Azure Cloud Service Target: " + resource.Id)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}
func (c AzureCloudServiceTargetConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c AzureCloudServiceTargetConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c AzureCloudServiceTargetConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.AzureCloudServiceResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if !c.isAzureCloudService(resource) {
		return nil
	}

	zap.L().Info("Azure Cloud Service Target: " + resource.Id)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c AzureCloudServiceTargetConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.AzureCloudServiceResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if !c.isAzureCloudService(resource) {
		return nil
	}

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + azureCloudServiceDeploymentDataType + "." + resourceName + ".deployment_targets[0].id}"
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

func (c AzureCloudServiceTargetConverter) buildData(resourceName string, resource octopus.AzureCloudServiceResource) terraform.TerraformDeploymentTargetsData {
	return terraform.TerraformDeploymentTargetsData{
		Type:        azureCloudServiceDeploymentDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c AzureCloudServiceTargetConverter) writeData(file *hclwrite.File, resource octopus.AzureCloudServiceResource, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c AzureCloudServiceTargetConverter) getLookup(stateless bool, targetName string) string {
	if stateless {
		return "${length(data." + azureCloudServiceDeploymentDataType + "." + targetName + ".deployment_targets) != 0 " +
			"? data." + azureCloudServiceDeploymentDataType + "." + targetName + ".deployment_targets[0].id " +
			": " + azureCloudServiceDeploymentResourceType + "." + targetName + "[0].id}"
	}
	return "${" + azureCloudServiceDeploymentResourceType + "." + targetName + ".id}"
}

func (c AzureCloudServiceTargetConverter) getDependency(stateless bool, targetName string) string {
	if stateless {
		return "${" + azureCloudServiceDeploymentResourceType + "." + targetName + "}"
	}

	return ""
}

func (c AzureCloudServiceTargetConverter) getCount(stateless bool, targetName string) *string {
	if stateless {
		return strutil.StrPointer("${length(data." + azureCloudServiceDeploymentDataType + "." + targetName + ".deployment_targets) != 0 ? 0 : 1}")
	}

	return nil
}

func (c AzureCloudServiceTargetConverter) toHcl(target octopus.AzureCloudServiceResource, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(target.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if c.isAzureCloudService(target) {
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
		thisResource.Name = target.Name
		thisResource.FileName = "space_population/" + targetName + ".tf"
		thisResource.Id = target.Id
		thisResource.ResourceType = c.GetResourceType()
		thisResource.Lookup = c.getLookup(stateless, targetName)
		thisResource.Dependency = c.getDependency(stateless, targetName)

		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformAzureCloudServiceDeploymentTarget{
				Type:                            azureCloudServiceDeploymentResourceType,
				Name:                            targetName,
				Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
				ResourceName:                    target.Name,
				Roles:                           target.Roles,
				AccountId:                       c.getAccount(target.Endpoint.AccountId, dependencies),
				CloudServiceName:                target.Endpoint.CloudServiceName,
				StorageAccountName:              target.Endpoint.StorageAccountName,
				DefaultWorkerPoolId:             &target.Endpoint.DefaultWorkerPoolId,
				HealthStatus:                    &target.HealthStatus,
				IsDisabled:                      &target.IsDisabled,
				MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId, dependencies),
				OperatingSystem:                 nil,
				ShellName:                       &target.ShellName,
				ShellVersion:                    &target.ShellVersion,
				Slot:                            nil,
				SpaceId:                         nil,
				Status:                          nil,
				StatusSummary:                   nil,
				SwapIfPossible:                  nil,
				TenantTags:                      c.Excluder.FilteredTenantTags(target.TenantTags, c.ExcludeTenantTags, c.ExcludeTenantTagSets),
				TenantedDeploymentParticipation: &target.TenantedDeploymentParticipation,
				Tenants:                         dependencies.GetResources("Tenants", target.TenantIds...),
				Thumbprint:                      &target.Thumbprint,
				Uri:                             nil,
				UseCurrentInstanceCount:         &target.Endpoint.UseCurrentInstanceCount,
				Endpoint: &terraform.TerraformAzureCloudServiceDeploymentTargetEndpoint{
					DefaultWorkerPoolId: c.getWorkerPool(target.Endpoint.DefaultWorkerPoolId, dependencies),
					CommunicationStyle:  "AzureCloudService",
				},
				Count: c.getCount(stateless, targetName),
			}
			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, target, targetName)
			}

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
	}

	return nil
}

func (c AzureCloudServiceTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c AzureCloudServiceTargetConverter) exportDependencies(target octopus.AzureCloudServiceResource, dependencies *data.ResourceDetailsCollection) error {

	// The machine policies need to be exported
	if err := c.MachinePolicyConverter.ToHclById(target.MachinePolicyId, dependencies); err != nil {
		return err
	}

	// Export the accounts
	if err := c.AccountConverter.ToHclById(target.Endpoint.AccountId, dependencies); err != nil {
		return err
	}

	// Export the environments
	for _, e := range target.EnvironmentIds {
		if err := c.EnvironmentConverter.ToHclById(e, dependencies); err != nil {
			return err
		}
	}

	return nil
}

func (c AzureCloudServiceTargetConverter) exportStatelessDependencies(target octopus.AzureCloudServiceResource, dependencies *data.ResourceDetailsCollection) error {

	// The machine policies need to be exported
	if err := c.MachinePolicyConverter.ToHclStatelessById(target.MachinePolicyId, dependencies); err != nil {
		return err
	}

	// Export the accounts
	if err := c.AccountConverter.ToHclStatelessById(target.Endpoint.AccountId, dependencies); err != nil {
		return err
	}

	// Export the environments
	for _, e := range target.EnvironmentIds {
		if err := c.EnvironmentConverter.ToHclStatelessById(e, dependencies); err != nil {
			return err
		}
	}

	return nil
}

func (c AzureCloudServiceTargetConverter) lookupEnvironments(envs []string, dependencies *data.ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c AzureCloudServiceTargetConverter) getMachinePolicy(machine string, dependencies *data.ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c AzureCloudServiceTargetConverter) getAccount(account string, dependencies *data.ResourceDetailsCollection) string {
	accountLookup := dependencies.GetResource("Accounts", account)
	if accountLookup == "" {
		return ""
	}

	return accountLookup
}

func (c AzureCloudServiceTargetConverter) getWorkerPool(pool string, dependencies *data.ResourceDetailsCollection) *string {
	if len(pool) == 0 {
		return nil
	}

	machineLookup := dependencies.GetResource("WorkerPools", pool)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}
