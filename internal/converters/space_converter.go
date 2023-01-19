package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/enhancedconverter"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

// SpaceConverter creates the files required to create a new space. These files are used in a separate
// terraform project, as you first need to a create a space, and then configure a second provider
// to use that space.
type SpaceConverter struct {
	Client client.OctopusClient
}

// ToHcl is a bulk export that takes advantage of the collection endpoints to download and export everything
// with no filter and with the least number of network calls. It is "dumb" in the sense that things like
// dependencies are manually resolved through the order in which resources are exported.
func (c SpaceConverter) ToHcl(dependencies *enhancedconverter.ResourceDetailsCollection) error {

	err := c.createSpaceTf(dependencies)

	if err != nil {
		return err
	}

	// Generate space population common files
	SpacePopulateCommonGenerator{}.ToHcl(dependencies)

	// Convert the feeds
	err = enhancedconverter.FeedConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the accounts
	err = enhancedconverter.AccountConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the environments
	err = enhancedconverter.EnvironmentConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the library variables
	err = enhancedconverter.LibraryVariableSetConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the lifecycles
	err = enhancedconverter.LifecycleConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the worker pools
	err = enhancedconverter.WorkerPoolConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the tag sets
	err = enhancedconverter.TagSetConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the git credentials
	err = enhancedconverter.GitCredentialsConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the projects groups
	err = enhancedconverter.ProjectGroupConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the projects
	err = enhancedconverter.ProjectConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the tenants
	err = enhancedconverter.TenantConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the certificates
	err = enhancedconverter.CertificateConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the tenant variables
	err = enhancedconverter.TenantVariableConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the machine policies
	err = enhancedconverter.MachinePolicyConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the k8s targets
	err = enhancedconverter.KubernetesTargetConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the ssh targets
	err = enhancedconverter.SshTargetConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the ssh targets
	err = enhancedconverter.ListeningTargetConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the polling targets
	err = enhancedconverter.PollingTargetConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the cloud region targets
	err = enhancedconverter.CloudRegionTargetConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the cloud region targets
	err = enhancedconverter.OfflineDropTargetConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the azure cloud service targets
	err = enhancedconverter.AzureCloudServiceTargetConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the azure cloud service targets
	err = enhancedconverter.AzureServiceFabricTargetConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the azure web app targets
	err = enhancedconverter.AzureWebAppTargetConverter{
		Client: c.Client,
	}.ToHcl(dependencies)

	if err != nil {
		return err
	}

	return nil
}

func (c SpaceConverter) getResourceType() string {
	return "Spaces"
}

func (c SpaceConverter) createSpaceTf(dependencies *enhancedconverter.ResourceDetailsCollection) error {
	space := octopus.Space{}
	err := c.Client.GetSpace(&space)

	if err != nil {
		return err
	}

	spaceResourceName := "octopus_space_" + util.SanitizeNamePointer(space.Name)
	spaceName := "${var.octopus_space_name}"

	thisResource := enhancedconverter.ResourceDetails{}
	thisResource.FileName = "space_creation/" + spaceResourceName + ".tf"
	thisResource.Id = space.Id
	thisResource.ResourceType = "Spaces"
	thisResource.Lookup = "${octopusdeploy_project." + spaceResourceName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformSpace{
			Description:        space.Description,
			IsDefault:          space.IsDefault,
			IsTaskQueueStopped: space.TaskQueueStopped,
			Name:               spaceResourceName,
			//SpaceManagersTeamMembers: space.SpaceManagersTeamMembers,
			//SpaceManagersTeams:       space.SpaceManagersTeams,
			// TODO: import teams rather than defaulting to admins
			SpaceManagersTeams: []string{"teams-administrators"},
			ResourceName:       &spaceName,
			Type:               "octopusdeploy_space",
		}

		spaceOutput := terraform.TerraformOutput{
			Name:  "octopus_space_id",
			Value: "${octopusdeploy_space." + spaceResourceName + ".id}",
		}

		spaceNameVar := terraform.TerraformVariable{
			Name:        "octopus_space_name",
			Type:        "string",
			Nullable:    false,
			Sensitive:   false,
			Description: "The name of the new space (the exported space was called " + *space.Name + ")",
			Default:     space.Name,
		}

		file := hclwrite.NewEmptyFile()
		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
		file.Body().AppendBlock(gohcl.EncodeAsBlock(spaceOutput, "output"))

		block := gohcl.EncodeAsBlock(spaceNameVar, "variable")
		util.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}
