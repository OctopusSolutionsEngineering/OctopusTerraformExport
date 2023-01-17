package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type AzureServiceFabricTargetConverter struct {
	Client            client.OctopusClient
	SpaceResourceName string
	MachinePolicyMap  map[string]string
	EnvironmentMap    map[string]string
	WorkerPoolMap     map[string]string
}

func (c AzureServiceFabricTargetConverter) ToHcl() (map[string]string, map[string]string, error) {
	collection := octopus.GeneralCollection[octopus.AzureServiceFabricResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return nil, nil, err
	}

	results := map[string]string{}
	resultsMap := map[string]string{}

	for _, target := range collection.Items {
		if target.Endpoint.CommunicationStyle == "AzureServiceFabricCluster" {
			targetName := "target_" + util.SanitizeName(target.Name)
			passwordLookup := "${var." + targetName + "}"

			terraformResource := terraform.TerraformAzureServiceFabricClusterDeploymentTarget{
				Type:                            "octopusdeploy_azure_service_fabric_cluster_deployment_target",
				Name:                            targetName,
				Environments:                    c.lookupEnvironments(target.EnvironmentIds),
				ResourceName:                    target.Name,
				Roles:                           target.Roles,
				ConnectionEndpoint:              target.Endpoint.ConnectionEndpoint,
				AadClientCredentialSecret:       &target.Endpoint.AadClientCredentialSecret,
				AadCredentialType:               &target.Endpoint.AadCredentialType,
				AadUserCredentialPassword:       &passwordLookup,
				AadUserCredentialUsername:       &target.Endpoint.AadUserCredentialUsername,
				CertificateStoreLocation:        &target.Endpoint.CertificateStoreLocation,
				CertificateStoreName:            &target.Endpoint.CertificateStoreName,
				ClientCertificateVariable:       &target.Endpoint.ClientCertVariable,
				HealthStatus:                    &target.HealthStatus,
				IsDisabled:                      &target.IsDisabled,
				MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId),
				OperatingSystem:                 nil,
				SecurityMode:                    nil,
				ServerCertificateThumbprint:     nil,
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
				Endpoint:                        nil,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			secretVariableResource := terraform.TerraformVariable{
				Name:        targetName,
				Type:        "string",
				Nullable:    true,
				Sensitive:   true,
				Description: "The aad_user_credential_password value associated with the target \"" + target.Name + "\"",
			}

			block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
			util.WriteUnquotedAttribute(block, "type", "string")
			file.Body().AppendBlock(block)

			results["space_population/target_"+targetName+".tf"] = string(file.Bytes())
			resultsMap[target.Id] = "${octopusdeploy_azure_service_fabric_cluster_deployment_target." + targetName + ".id}"
		}
	}

	return results, resultsMap, nil
}

func (c AzureServiceFabricTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c AzureServiceFabricTargetConverter) lookupEnvironments(envs []string) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = c.EnvironmentMap[v]
	}
	return newEnvs
}

func (c AzureServiceFabricTargetConverter) getMachinePolicy(machine string) *string {
	machineLookup, ok := c.MachinePolicyMap[machine]
	if !ok {
		return nil
	}

	return &machineLookup
}

func (c AzureServiceFabricTargetConverter) getWorkerPool(pool string) *string {
	machineLookup, ok := c.WorkerPoolMap[pool]
	if !ok {
		return nil
	}

	return &machineLookup
}
