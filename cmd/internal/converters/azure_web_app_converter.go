package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
)

type AzureWebAppTargetConverter struct {
	Client                 client.OctopusClient
	MachinePolicyConverter ConverterById
	AccountConverter       ConverterById
	EnvironmentConverter   ConverterById
}

func (c AzureWebAppTargetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.AzureWebAppResource]{}
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

func (c AzureWebAppTargetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.AzureWebAppResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, dependencies)
}

func (c AzureWebAppTargetConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.AzureWebAppResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	thisResource := ResourceDetails{}

	resourceName := "target_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${data.octopusdeploy_deployment_targets" + resourceName + ".deployment_targets[0].id}"
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
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c AzureWebAppTargetConverter) toHcl(target octopus2.AzureWebAppResource, recursive bool, dependencies *ResourceDetailsCollection) error {

	if target.Endpoint.CommunicationStyle == "AzureWebApp" {

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
		thisResource.Lookup = "${octopusdeploy_azure_web_app_deployment_target." + targetName + ".id}"
		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformAzureWebAppDeploymentTarget{
				Type:                            "octopusdeploy_azure_web_app_deployment_target",
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
				TenantTags:                      target.TenantTags,
				TenantedDeploymentParticipation: &target.TenantedDeploymentParticipation,
				Tenants:                         target.TenantIds,
				Thumbprint:                      &target.Thumbprint,
				Uri:                             nil,
				WebAppSlotName:                  &target.Endpoint.WebAppSlotName,
				Endpoint: &terraform.TerraformAzureWebAppDeploymentTargetEndpoint{
					DefaultWorkerPoolId: c.getWorkerPool(target.Endpoint.DefaultWorkerPoolId, dependencies),
					CommunicationStyle:  "AzureWebApp",
				},
			}
			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + target.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_azure_web_app_deployment_target." + targetName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}

		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c AzureWebAppTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c AzureWebAppTargetConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c AzureWebAppTargetConverter) getMachinePolicy(machine string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c AzureWebAppTargetConverter) getAccount(account string, dependencies *ResourceDetailsCollection) string {
	accountLookup := dependencies.GetResource("Accounts", account)
	if accountLookup == "" {
		return ""
	}

	return accountLookup
}

func (c AzureWebAppTargetConverter) getWorkerPool(pool string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("WorkerPools", pool)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c AzureWebAppTargetConverter) exportDependencies(target octopus2.AzureWebAppResource, dependencies *ResourceDetailsCollection) error {

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
