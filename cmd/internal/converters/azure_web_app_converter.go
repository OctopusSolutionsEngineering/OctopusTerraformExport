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

const octopusdeployAzureWebAppDeploymentTargetDataType = "octopusdeploy_deployment_targets"
const octopusdeployAzureWebAppDeploymentTargetResourceType = "octopusdeploy_azure_web_app_deployment_target"

type AzureWebAppTargetConverter struct {
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
}

func (c AzureWebAppTargetConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) error {
	return c.allToHcl(false, dependencies)
}

func (c AzureWebAppTargetConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) error {
	return c.allToHcl(true, dependencies)
}

func (c AzureWebAppTargetConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	collection := octopus.GeneralCollection[octopus.AzureWebAppResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	targets := lo.Filter(collection.Items, func(item octopus.AzureWebAppResource, index int) bool {
		return c.isAzureWebApp(item)
	})

	for _, resource := range targets {
		zap.L().Info("Azure Web App Target: " + resource.Id)
		err = c.toHcl(resource, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c AzureWebAppTargetConverter) isAzureWebApp(resource octopus.AzureWebAppResource) bool {
	return resource.Endpoint.CommunicationStyle == "AzureWebApp"
}

func (c AzureWebAppTargetConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c AzureWebAppTargetConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c AzureWebAppTargetConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.AzureWebAppResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if !c.isAzureWebApp(resource) {
		return nil
	}

	zap.L().Info("Azure Web App Target: " + resource.Id)
	return c.toHcl(resource, true, stateless, dependencies)
}

func (c AzureWebAppTargetConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.AzureWebAppResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(resource.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if !c.isAzureWebApp(resource) {
		return nil
	}

	thisResource := data.ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data." + octopusdeployAzureWebAppDeploymentTargetDataType + "." + resourceName + ".deployment_targets[0].id}"
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

func (c AzureWebAppTargetConverter) buildData(resourceName string, resource octopus.AzureWebAppResource) terraform.TerraformDeploymentTargetsData {
	return terraform.TerraformDeploymentTargetsData{
		Type:        octopusdeployAzureWebAppDeploymentTargetDataType,
		Name:        resourceName,
		Ids:         nil,
		PartialName: &resource.Name,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c AzureWebAppTargetConverter) writeData(file *hclwrite.File, resource octopus.AzureWebAppResource, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c AzureWebAppTargetConverter) toHcl(target octopus.AzureWebAppResource, recursive bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded targets
	if c.Excluder.IsResourceExcludedWithRegex(target.Name, c.ExcludeAllTargets, c.ExcludeTargets, c.ExcludeTargetsRegex, c.ExcludeTargetsExcept) {
		return nil
	}

	if !c.isAzureWebApp(target) {
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
	thisResource.Name = target.Name
	thisResource.FileName = "space_population/" + targetName + ".tf"
	thisResource.Id = target.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${" + octopusdeployAzureWebAppDeploymentTargetResourceType + "." + targetName + ".id}"

	if stateless {
		thisResource.Lookup = "${length(data." + octopusdeployAzureWebAppDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 " +
			"? data." + octopusdeployAzureWebAppDeploymentTargetDataType + "." + targetName + ".deployment_targets[0].id " +
			": " + octopusdeployAzureWebAppDeploymentTargetResourceType + "." + targetName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployAzureWebAppDeploymentTargetResourceType + "." + targetName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployAzureWebAppDeploymentTargetResourceType + "." + targetName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformAzureWebAppDeploymentTarget{
			Type:                            octopusdeployAzureWebAppDeploymentTargetResourceType,
			Name:                            targetName,
			Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
			ResourceName:                    target.Name,
			Roles:                           target.Roles,
			AccountId:                       c.getAccount(target.Endpoint.AccountId, dependencies),
			ResourceGroupName:               target.Endpoint.ResourceGroupName,
			WebAppName:                      target.Endpoint.WebAppName,
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
			Thumbprint:                      &target.Thumbprint,
			Uri:                             nil,
			WebAppSlotName:                  &target.Endpoint.WebAppSlotName,
			Endpoint: &terraform.TerraformAzureWebAppDeploymentTargetEndpoint{
				DefaultWorkerPoolId: c.getWorkerPool(target.Endpoint.DefaultWorkerPoolId, dependencies),
				CommunicationStyle:  "AzureWebApp",
			},
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, target, targetName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployAzureWebAppDeploymentTargetDataType + "." + targetName + ".deployment_targets) != 0 ? 0 : 1}")
		}

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), target.Name, octopusdeployAzureWebAppDeploymentTargetResourceType, targetName))

		targetBlock := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(targetBlock)
		}

		err := TenantTagDependencyGenerator{}.AddAndWriteTagSetDependencies(c.Client, terraformResource.TenantTags, c.TagSetConverter, targetBlock, dependencies, recursive)
		if err != nil {
			return "", err
		}
		file.Body().AppendBlock(targetBlock)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c AzureWebAppTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c AzureWebAppTargetConverter) lookupEnvironments(envs []string, dependencies *data.ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c AzureWebAppTargetConverter) getMachinePolicy(machine string, dependencies *data.ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c AzureWebAppTargetConverter) getAccount(account string, dependencies *data.ResourceDetailsCollection) string {
	accountLookup := dependencies.GetResource("Accounts", account)
	if accountLookup == "" {
		return ""
	}

	return accountLookup
}

func (c AzureWebAppTargetConverter) getWorkerPool(pool string, dependencies *data.ResourceDetailsCollection) *string {
	if len(pool) == 0 {
		return nil
	}

	machineLookup := dependencies.GetResource("WorkerPools", pool)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c AzureWebAppTargetConverter) exportDependencies(target octopus.AzureWebAppResource, dependencies *data.ResourceDetailsCollection) error {

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

func (c AzureWebAppTargetConverter) exportStatelessDependencies(target octopus.AzureWebAppResource, dependencies *data.ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclStatelessById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the accounts
	err = c.AccountConverter.ToHclStatelessById(target.Endpoint.AccountId, dependencies)

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
