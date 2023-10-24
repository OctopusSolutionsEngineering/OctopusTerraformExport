package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
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
}

// ToHcl is a bulk export that takes advantage of the collection endpoints to download and export everything
// with no filter and with the least number of network calls.
func (c SpaceConverter) ToHcl(dependencies *ResourceDetailsCollection) error {

	err := c.createSpaceTf(dependencies)

	if err != nil {
		return err
	}

	// Convert the feeds
	err = c.FeedConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the accounts
	err = c.AccountConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the environments
	err = c.EnvironmentConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the library variables
	err = c.LibraryVariableSetConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the lifecycles
	err = c.LifecycleConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the worker pools
	err = c.WorkerPoolConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the tag sets
	err = c.TagSetConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the git credentials
	err = c.GitCredentialsConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the projects groups
	err = c.ProjectGroupConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the projects
	err = c.ProjectConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the tenants
	err = c.TenantConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the certificates
	err = c.CertificateConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the tenant variables
	err = c.TenantVariableConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the machine policies
	err = c.MachinePolicyConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the k8s targets
	err = c.KubernetesTargetConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the ssh targets
	err = c.SshTargetConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the ssh targets
	err = c.ListeningTargetConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the polling targets
	err = c.PollingTargetConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the cloud region targets
	err = c.CloudRegionTargetConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the cloud region targets
	err = c.OfflineDropTargetConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the azure cloud service targets
	err = c.AzureCloudServiceTargetConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the azure cloud service targets
	err = c.AzureServiceFabricTargetConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	// Convert the azure web app targets
	err = c.AzureWebAppTargetConverter.ToHcl(dependencies)

	if err != nil {
		return err
	}

	return nil
}

func (c SpaceConverter) getResourceType() string {
	return "Spaces"
}

func (c SpaceConverter) createSpaceTf(dependencies *ResourceDetailsCollection) error {
	space := octopus.Space{}
	err := c.Client.GetSpace(&space)

	if err != nil {
		return err
	}

	spaceResourceName := "octopus_space_" + sanitizer.SanitizeName(space.Name)
	spaceName := "${var.octopus_space_name}"

	thisResource := ResourceDetails{}
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
			SpaceManagersTeams: []string{"teams-administrators"},
			ResourceName:       &spaceName,
			Type:               "octopusdeploy_space",
		}

		spaceOutput := terraform2.TerraformOutput{
			Name:  "octopus_space_id",
			Value: "${octopusdeploy_space." + spaceResourceName + ".id}",
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
		file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
			Type: hclsyntax.TokenComment,
			Bytes: []byte("# Import existing resources with the following commands:\n" +
				"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/Spaces | jq -r '.Items[] | select(.Name==\"" + space.Name + "\") | .Id')\n" +
				"# terraform import octopusdeploy_space." + spaceName + " ${RESOURCE_ID}\n"),
			SpacesBefore: 0,
		}})

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))
		file.Body().AppendBlock(gohcl.EncodeAsBlock(spaceOutput, "output"))

		block := gohcl.EncodeAsBlock(spaceNameVar, "variable")
		hcl.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)
	return nil
}
