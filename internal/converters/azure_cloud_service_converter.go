package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type AzureCloudServiceTargetConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	MachinePolicyMap  map[string]string
	EnvironmentMap    map[string]string
	AccountMap        map[string]string
}

func (c AzureCloudServiceTargetConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.AzureCloudServiceResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	for _, target := range collection.Items {
		if target.Endpoint.CommunicationStyle == "AzureCloudService" {
			targetName := "target_" + util.SanitizeName(target.Name)

			terraformResource := terraform.TerraformAzureCloudServiceDeploymentTarget{
				Type:                            "octopusdeploy_azure_cloud_service_deployment_target",
				Name:                            targetName,
				Environments:                    c.lookupEnvironments(target.EnvironmentIds),
				ResourceName:                    target.Name,
				Roles:                           target.Roles,
				AccountId:                       c.getAccount(target.Endpoint.AccountId),
				CloudServiceName:                target.Endpoint.CloudServiceName,
				StorageAccountName:              target.Endpoint.StorageAccountName,
				DefaultWorkerPoolId:             &target.Endpoint.DefaultWorkerPoolId,
				HealthStatus:                    &target.HealthStatus,
				IsDisabled:                      &target.IsDisabled,
				MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId),
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
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			results["space_population/target_"+targetName+".tf"] = string(file.Bytes())
			resultsMap[target.Id] = "${octopusdeploy_azure_cloud_service_deployment_target." + targetName + ".id}"
		}
	}

	return results, resultsMap, nil
}

func (c AzureCloudServiceTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c AzureCloudServiceTargetConverter) lookupEnvironments(envs []string) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = c.EnvironmentMap[v]
	}
	return newEnvs
}

func (c AzureCloudServiceTargetConverter) getMachinePolicy(machine string) *string {
	machineLookup, ok := c.MachinePolicyMap[machine]
	if !ok {
		return nil
	}

	return &machineLookup
}

func (c AzureCloudServiceTargetConverter) getAccount(account string) string {
	accountLookup, ok := c.AccountMap[account]
	if !ok {
		return ""
	}

	return accountLookup
}
