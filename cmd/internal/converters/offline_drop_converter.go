package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
)

type OfflineDropTargetConverter struct {
	Client                 client.OctopusClient
	MachinePolicyConverter ConverterById
	EnvironmentConverter   ConverterById
}

func (c OfflineDropTargetConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.OfflineDropResource]{}
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

func (c OfflineDropTargetConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.OfflineDropResource{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, true, dependencies)
}

func (c OfflineDropTargetConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
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

	if resource.Endpoint.CommunicationStyle != "OfflineDrop" {
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
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c OfflineDropTargetConverter) toHcl(target octopus.OfflineDropResource, recursive bool, dependencies *ResourceDetailsCollection) error {
	if target.Endpoint.CommunicationStyle == "OfflineDrop" {
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
		thisResource.Lookup = "${octopusdeploy_offline_package_drop_deployment_target." + targetName + ".id}"
		thisResource.ToHcl = func() (string, error) {

			terraformResource := terraform.TerraformOfflineDropDeploymentTarget{
				Type:                            "octopusdeploy_offline_package_drop_deployment_target",
				Name:                            targetName,
				ApplicationsDirectory:           target.Endpoint.ApplicationsDirectory,
				WorkingDirectory:                target.Endpoint.OctopusWorkingDirectory,
				Environments:                    c.lookupEnvironments(target.EnvironmentIds, dependencies),
				ResourceName:                    target.Name,
				Roles:                           target.Roles,
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
				Thumbprint:                      nil,
				Uri:                             nil,
			}
			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + target.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_offline_package_drop_deployment_target." + targetName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}

		dependencies.AddResource(thisResource)
	}

	return nil
}

func (c OfflineDropTargetConverter) GetResourceType() string {
	return "Machines"
}

func (c OfflineDropTargetConverter) lookupEnvironments(envs []string, dependencies *ResourceDetailsCollection) []string {
	newEnvs := make([]string, len(envs))
	for i, v := range envs {
		newEnvs[i] = dependencies.GetResource("Environments", v)
	}
	return newEnvs
}

func (c OfflineDropTargetConverter) getMachinePolicy(machine string, dependencies *ResourceDetailsCollection) *string {
	machineLookup := dependencies.GetResource("MachinePolicies", machine)
	if machineLookup == "" {
		return nil
	}

	return &machineLookup
}

func (c OfflineDropTargetConverter) exportDependencies(target octopus.OfflineDropResource, dependencies *ResourceDetailsCollection) error {

	// The machine policies need to be exported
	err := c.MachinePolicyConverter.ToHclById(target.MachinePolicyId, dependencies)

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
