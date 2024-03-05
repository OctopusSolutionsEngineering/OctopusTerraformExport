package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"golang.org/x/sync/errgroup"
)

// SpaceConverter creates the files required to create a new space. These files are used in a separate
// terraform project, as you first need to a create a space, and then configure a second provider
// to use that space.
type SpaceConverter struct {
	Client                            client.OctopusClient
	AccountConverter                  Converter
	FeedConverter                     Converter
	EnvironmentConverter              Converter
	LibraryVariableSetConverter       Converter
	LifecycleConverter                Converter
	WorkerPoolConverter               Converter
	TagSetConverter                   Converter
	GitCredentialsConverter           Converter
	ProjectGroupConverter             Converter
	ProjectConverter                  Converter
	TenantConverter                   Converter
	CertificateConverter              Converter
	TenantVariableConverter           Converter
	MachinePolicyConverter            Converter
	KubernetesTargetConverter         Converter
	SshTargetConverter                Converter
	ListeningTargetConverter          Converter
	PollingTargetConverter            Converter
	CloudRegionTargetConverter        Converter
	OfflineDropTargetConverter        Converter
	AzureCloudServiceTargetConverter  Converter
	AzureServiceFabricTargetConverter Converter
	AzureWebAppTargetConverter        Converter
	ErrGroup                          *errgroup.Group
}

// AllToHcl is a bulk export that takes advantage of the collection endpoints to download and export everything
// with no filter and with the least number of network calls.
func (c SpaceConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) error {

	err := c.createSpaceTf(dependencies)

	if err != nil {
		return err
	}

	// Convert the feeds
	c.FeedConverter.AllToHcl(dependencies)

	// Convert the accounts
	c.AccountConverter.AllToHcl(dependencies)

	// Convert the environments
	c.EnvironmentConverter.AllToHcl(dependencies)

	// Convert the library variables
	c.LibraryVariableSetConverter.AllToHcl(dependencies)

	// Convert the lifecycles
	c.LifecycleConverter.AllToHcl(dependencies)

	// Convert the worker pools
	c.WorkerPoolConverter.AllToHcl(dependencies)

	// Convert the tag sets
	c.TagSetConverter.AllToHcl(dependencies)

	// Convert the git credentials
	c.GitCredentialsConverter.AllToHcl(dependencies)

	// Convert the projects groups
	c.ProjectGroupConverter.AllToHcl(dependencies)

	// Convert the projects
	c.ProjectConverter.AllToHcl(dependencies)

	// Convert the tenants
	c.TenantConverter.AllToHcl(dependencies)

	// Convert the certificates
	c.CertificateConverter.AllToHcl(dependencies)

	// Convert the tenant variables
	c.TenantVariableConverter.AllToHcl(dependencies)

	// Convert the machine policies
	c.MachinePolicyConverter.AllToHcl(dependencies)

	// Convert the k8s targets
	c.KubernetesTargetConverter.AllToHcl(dependencies)

	// Convert the ssh targets
	c.SshTargetConverter.AllToHcl(dependencies)

	// Convert the ssh targets
	c.ListeningTargetConverter.AllToHcl(dependencies)

	// Convert the polling targets
	c.PollingTargetConverter.AllToHcl(dependencies)

	// Convert the cloud region targets
	c.CloudRegionTargetConverter.AllToHcl(dependencies)

	// Convert the cloud region targets
	c.OfflineDropTargetConverter.AllToHcl(dependencies)

	// Convert the azure cloud service targets
	c.AzureCloudServiceTargetConverter.AllToHcl(dependencies)

	// Convert the azure cloud service targets
	c.AzureServiceFabricTargetConverter.AllToHcl(dependencies)

	// Convert the azure web app targets
	c.AzureWebAppTargetConverter.AllToHcl(dependencies)

	return c.ErrGroup.Wait()
}

// AllToStatelessHcl is a bulk export that takes advantage of the collection endpoints to download and export everything
// with no filter and with the least number of network calls. The exported module is suitable for a stateless terraform
// apply.
func (c SpaceConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) error {

	// Convert the feeds
	err := c.FeedConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the accounts
	err = c.AccountConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the environments
	err = c.EnvironmentConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the library variables
	err = c.LibraryVariableSetConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the lifecycles
	err = c.LifecycleConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the worker pools
	err = c.WorkerPoolConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the tag sets
	err = c.TagSetConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the git credentials
	err = c.GitCredentialsConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the projects groups
	err = c.ProjectGroupConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the projects
	err = c.ProjectConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the tenants
	err = c.TenantConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the certificates
	err = c.CertificateConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the tenant variables
	err = c.TenantVariableConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the machine policies
	err = c.MachinePolicyConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the k8s targets
	err = c.KubernetesTargetConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the ssh targets
	err = c.SshTargetConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the ssh targets
	err = c.ListeningTargetConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the polling targets
	err = c.PollingTargetConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the cloud region targets
	err = c.CloudRegionTargetConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the cloud region targets
	err = c.OfflineDropTargetConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the azure cloud service targets
	err = c.AzureCloudServiceTargetConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the azure cloud service targets
	err = c.AzureServiceFabricTargetConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the azure web app targets
	err = c.AzureWebAppTargetConverter.AllToStatelessHcl(dependencies)

	if err != nil {
		return err
	}

	return nil
}

func (c SpaceConverter) getResourceType() string {
	return "Spaces"
}

func (c SpaceConverter) createSpaceTf(dependencies *data.ResourceDetailsCollection) error {
	space := octopus.Space{}
	err := c.Client.GetSpace(&space)

	if err != nil {
		return err
	}

	spaceResourceName := "octopus_space_" + sanitizer.SanitizeName(space.Name)
	spaceName := "${var.octopus_space_name}"

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_creation/" + spaceResourceName + ".tf"
	thisResource.Id = space.Id
	thisResource.ResourceType = "Spaces"
	thisResource.Lookup = "${octopusdeploy_space." + spaceResourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform2.TerraformSpace{
			Description:        space.Description,
			IsDefault:          space.IsDefault,
			IsTaskQueueStopped: space.TaskQueueStopped,
			Name:               spaceResourceName,
			//SpaceManagersTeamMembers: space.SpaceManagersTeamMembers,
			//SpaceManagersTeams:       space.SpaceManagersTeams,
			// TODO: import teams rather than defaulting to admins
			SpaceManagersTeams: []string{"${var.octopus_space_managers}"},
			ResourceName:       &spaceName,
			Type:               "octopusdeploy_space",
		}

		defaultSpaceManagers := "teams-administrators"
		spaceManagerTeams := terraform2.TerraformVariable{
			Name:        "octopus_space_managers",
			Type:        "string",
			Nullable:    false,
			Sensitive:   false,
			Description: "The space manager teams for the new space",
			Default:     &defaultSpaceManagers,
		}

		spaceOutput := terraform2.TerraformOutput{
			Name:  "octopus_space_id",
			Value: "${octopusdeploy_space." + spaceResourceName + ".id}",
		}

		octopusSpaceName := terraform2.TerraformOutput{
			Name:  "octopus_space_name",
			Value: "${var.octopus_space_name}",
		}

		spaceNameVar := terraform2.TerraformVariable{
			Name:        "octopus_space_name",
			Type:        "string",
			Nullable:    false,
			Sensitive:   false,
			Description: "The name of the new space (the exported space was called " + space.Name + ")",
			Default:     &space.Name,
		}

		file := hclwrite.NewEmptyFile()

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, "Spaces", space.Name, "octopusdeploy_space", spaceName))

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
		file.Body().AppendBlock(gohcl.EncodeAsBlock(spaceOutput, "output"))
		file.Body().AppendBlock(gohcl.EncodeAsBlock(octopusSpaceName, "output"))

		block := gohcl.EncodeAsBlock(spaceNameVar, "variable")
		hcl.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		spaceManagerTeamsBlock := gohcl.EncodeAsBlock(spaceManagerTeams, "variable")
		hcl.WriteUnquotedAttribute(spaceManagerTeamsBlock, "type", "string")
		file.Body().AppendBlock(spaceManagerTeamsBlock)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}
