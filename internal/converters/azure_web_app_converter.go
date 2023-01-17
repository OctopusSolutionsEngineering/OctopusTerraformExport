package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type AzureWebAppTargetConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	MachinePolicyMap  map[string]string
	EnvironmentMap    map[string]string
	AccountMap        map[string]string
	WorkerPoolMap     map[string]string
}

func (c AzureWebAppTargetConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.AzureWebAppResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	for _, target := range collection.Items {
		if target.Endpoint.CommunicationStyle == "AzureWebApp" {
			targetName := "target_" + util.SanitizeName(target.Name)

			terraformResource := terraform.TerraformAzureWebAppDeploymentTarget{
				Type:                            "octopusdeploy_azure_web_app_deployment_target",
				Name:                            targetName,
				Environments:                    c.lookupEnvironments(target.EnvironmentIds),
				ResourceName:                    target.Name,
				Roles:                           target.Roles,
				AccountId:                       c.getAccount(target.Endpoint.AccountId),
				ResourceGroupName:               target.Endpoint.ResourceGroupName,
				WebAppName:                      target.Endpoint.WebAppName,
				HealthStatus:                    &target.HealthStatus,
				IsDisabled:                      &target.IsDisabled,
				MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId),
				OperatingSystem:                 nil,
				ShellName:                       &target.ShellName,
				ShellVersion:                    &target.ShellVersion,
				SpaceId:                         nil,
				Status:                          nil,
				StatusSummary:                   nil,
				TenantTags:                      target.TenantTags,
				TenantedDeploymentParticipation: &target.TenantedDeploymentParticipation,
				Tenants:                         target.TenantIds,
				Thumbprint:                      &target.Thumbprint,
				Uri:                             nil,
				WebAppSlotName:                  &target.Endpoint.WebAppSlotName,
				Endpoint: terraform.TerraformAzureWebAppDeploymentTargetEndpoint{
					DefaultWorkerPoolId: c.getWorkerPool(target.Endpoint.DefaultWorkerPoolId),
					CommunicationStyle:  "AzureWebApp",
				},
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			results["space_population/target_"+targetName+".tf"] = string(file.Bytes())
			resultsMap[target.Id] = "${octopusdeploy_azure_web_app_deployment_target." + targetName + ".id}"
		}
	}

	return results, resultsMap, nil
}

func (c AzureWebAppTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c AzureWebAppTargetConverter) lookupEnvironments(envs []string) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = c.EnvironmentMap[v]
	}
	return newEnvs
}

func (c AzureWebAppTargetConverter) getMachinePolicy(machine string) *string {
	machineLookup, ok := c.MachinePolicyMap[machine]
	if !ok {
		return nil
	}

	return &machineLookup
}

func (c AzureWebAppTargetConverter) getAccount(account string) string {
	accountLookup, ok := c.AccountMap[account]
	if !ok {
		return ""
	}

	return accountLookup
}

func (c AzureWebAppTargetConverter) getWorkerPool(pool string) *string {
	machineLookup, ok := c.WorkerPoolMap[pool]
	if !ok {
		return nil
	}

	return &machineLookup
}
