package enhancedconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type AzureServiceFabricTargetConverter struct {
	Client client.OctopusClient
}

func (c AzureServiceFabricTargetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.AzureServiceFabricResource]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c AzureServiceFabricTargetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	resource := octopus.AzureServiceFabricResource{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, dependencies)
}

func (c AzureServiceFabricTargetConverter) toHcl(target octopus.AzureServiceFabricResource, recursive bool, dependencies *ResourceDetailsCollection) error {
	if recursive {
		err := c.exportDependencies(target, dependencies)

		if err != nil {
			return err
		}
	}

	targetName := "target_" + util.SanitizeName(target.Name)

	thisResource := ResourceDetails{}
	thisResource.FileName = "space_population/" + targetName + ".tf"
	thisResource.Id = target.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_azure_service_fabric_cluster_deployment_target." + targetName + ".id}"
	thisResource.ToHcl = func() (string, error) {
		if target.Endpoint.CommunicationStyle == "AzureServiceFabricCluster" {

			passwordLookup := "${var." + targetName + "}"

			terraformResource := terraform.TerraformAzureServiceFabricClusterDeploymentTarget{
				Type:                            "octopusdeploy_azure_service_fabric_cluster_deployment_target",
				Name:                            targetName,
				Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
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
				MachinePolicyId:                 c.getMachinePolicy(target.MachinePolicyId, dependencies),
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

			return string(file.Bytes()), nil
		}

		return "", nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c AzureServiceFabricTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c AzureServiceFabricTargetConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c AzureServiceFabricTargetConverter) getMachinePolicy(machine string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c AzureServiceFabricTargetConverter) getWorkerPool(pool string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("WorkerPools", pool)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c AzureServiceFabricTargetConverter) exportDependencies(target octopus.AzureServiceFabricResource, dependencies *ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := MachinePolicyConverter{
		Client: c.Client,
	}.ToHclById(target.MachinePolicyId, dependencies)

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

	return nil
}
