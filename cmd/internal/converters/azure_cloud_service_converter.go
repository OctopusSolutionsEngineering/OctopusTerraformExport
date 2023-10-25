package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

type AzureCloudServiceTargetConverter struct {
	Client                 client.OctopusClient
	MachinePolicyConverter ConverterById
	AccountConverter       ConverterById
	EnvironmentConverter   ConverterById
	ExcludeAllTargets      bool
	ExcludeTenantTags      args.ExcludeTenantTags
	ExcludeTenantTagSets   args.ExcludeTenantTagSets
	Excluder               ExcludeByName
}

func (c AzureCloudServiceTargetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.AzureCloudServiceResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Azure Cloud Service Target: " + resource.Id)
		err = c.toHcl(resource, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c AzureCloudServiceTargetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
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

	zap.L().Info("Azure Cloud Service Target: " + resource.Id)
	return c.toHcl(resource, true, dependencies)
}

func (c AzureCloudServiceTargetConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.Machine{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	if resource.Endpoint.CommunicationStyle != "AzureCloudService" {
		return nil
	}

	thisResource := ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data.octopusdeploy_deployment_targets." + resourceName + ".deployment_targets[0].id}"
	thisResource.ToHcl = func() (string, error) {
		terraformResource := terraform.TerraformDeploymentTargetsData{
			Type:        "octopusdeploy_deployment_targets",
			Name:        resourceName,
			Ids:         nil,
			PartialName: &resource.Name,
			Skip:        0,
			Take:        1,
		}
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve a deployment target called \""+resource.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(self.deployment_targets) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c AzureCloudServiceTargetConverter) toHcl(target octopus.AzureCloudServiceResource, recursive bool, dependencies *ResourceDetailsCollection) error {
	if c.ExcludeAllTargets {
		return nil
	}

	if target.Endpoint.CommunicationStyle == "AzureCloudService" {
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
		thisResource.Lookup = "${octopusdeploy_azure_cloud_service_deployment_target." + targetName + ".id}"
		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformAzureCloudServiceDeploymentTarget{
				Type:                            "octopusdeploy_azure_cloud_service_deployment_target",
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
			}
			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + target.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_azure_cloud_service_deployment_target." + targetName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}

		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c AzureCloudServiceTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c AzureCloudServiceTargetConverter) exportDependencies(target octopus.AzureCloudServiceResource, dependencies *ResourceDetailsCollection) error {

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

func (c AzureCloudServiceTargetConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c AzureCloudServiceTargetConverter) getMachinePolicy(machine string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c AzureCloudServiceTargetConverter) getAccount(account string, dependencies *ResourceDetailsCollection) string {
	accountLookup := dependencies.GetResource("Accounts", account)
	if accountLookup == "" {
		return ""
	}

	return accountLookup
}

func (c AzureCloudServiceTargetConverter) getWorkerPool(pool string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("WorkerPools", pool)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}
