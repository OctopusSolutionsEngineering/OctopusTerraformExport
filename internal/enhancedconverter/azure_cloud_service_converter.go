package enhancedconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type AzureCloudServiceTargetConverter struct {
	Client client.OctopusClient
}

func (c AzureCloudServiceTargetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.AzureCloudServiceResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c AzureCloudServiceTargetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	resource := octopus.AzureCloudServiceResource{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, dependencies)
}

func (c AzureCloudServiceTargetConverter) toHcl(target octopus.AzureCloudServiceResource, dependencies *ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := MachinePolicyConverter{
		Client: c.Client,
	}.ToHclById(target.MachinePolicyId, dependencies)

	if err != nil {
		return err
	}

	// Export the accounts
	err = AccountConverter{
		Client: c.Client,
	}.ToHclById(target.Endpoint.AccountId, dependencies)

	if err != nil {
		return err
	}

	// Export the environments
	for _, e := range target.EnvironmentIds {
		err = EnvironmentConverter{
			Client: c.Client,
		}.ToHclById(e, dependencies)

		if err != nil {
			return err
		}
	}

	targetName := "target_" + util.SanitizeName(target.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + targetName + ".tf"
	thisResource.Id = target.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_project." + targetName + ".id}"
	thisResource.ToHcl = func() (string, error) {
		if target.Endpoint.CommunicationStyle == "AzureCloudService" {
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
				TenantTags:                      target.TenantTags,
				TenantedDeploymentParticipation: &target.TenantedDeploymentParticipation,
				Tenants:                         target.TenantIds,
				Thumbprint:                      &target.Thumbprint,
				Uri:                             nil,
				UseCurrentInstanceCount:         &target.Endpoint.UseCurrentInstanceCount,
				Endpoint: &terraform.TerraformAzureCloudServiceDeploymentTargetEndpoint{
					DefaultWorkerPoolId: c.getWorkerPool(target.Endpoint.DefaultWorkerPoolId, dependencies),
					CommunicationStyle:  "AzureCloudService",
				},
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}

		return "", nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c AzureCloudServiceTargetConverter) GetResourceType() string {
	return "Machines"
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
