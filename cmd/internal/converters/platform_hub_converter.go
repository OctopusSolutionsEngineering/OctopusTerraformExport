package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"golang.org/x/sync/errgroup"
)

type PlatformHubConverter struct {
	Client                    client.OctopusClient
	ErrGroup                  *errgroup.Group
	DummySecretVariableValues bool
	DummySecretGenerator      dummy.DummySecretGenerator
}

func (c PlatformHubConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c PlatformHubConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c PlatformHubConverter) allToHcl(all bool, dependencies *data.ResourceDetailsCollection) error {
	resource := &octopus.OctopusPlatformHubVersionControlUsernamePasswordSetting{}
	if err := c.Client.GetAllGlobalResources("PlatformHub/VersionControl", &resource, []string{}, []string{}); err != nil {
		return err
	}

	// Assume if no URL is set, there is nothing to export
	if resource.Url == "" {
		return nil
	}

	/*
		The platform hub version control settings are unique in that there is only one setting across the entire space.
		It has no name or ID.
	*/
	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/platform_hub_version_control.tf"
	thisResource.Id = "PlatformHubVersionControl"
	thisResource.Name = "PlatformHubVersionControl"
	thisResource.ResourceType = "PlatformHubVersionControl"

	thisResource.ToHcl = func() (string, error) {
		if resource.Credentials.Type == "UsernamePassword" {
			return c.generateUsernamePasswordHcl(resource, dependencies)
		}

		return "", nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c PlatformHubConverter) generateUsernamePasswordHcl(resource *octopus.OctopusPlatformHubVersionControlUsernamePasswordSetting, dependencies *data.ResourceDetailsCollection) (string, error) {
	terraformResource := terraform.TerraformPlatformHubVersionControlUsernamePasswordSetting{
		Type:          "octopusdeploy_platform_hub_version_control_username_password_settings",
		Name:          "PlatformHubVersionControl",
		Count:         nil,
		Url:           resource.Url,
		DefaultBranch: resource.DefaultBranch,
		BasePath:      resource.BasePath,
		Username:      "${var.PlatformHubVersionControlUsername}",
		Password:      "${var.PlatformHubVersionControlPassword}",
	}
	file := hclwrite.NewEmptyFile()
	block := gohcl.EncodeAsBlock(terraformResource, "resource")
	file.Body().AppendBlock(block)

	usernameVariableResource := terraform.TerraformVariable{
		Name:        "PlatformHubVersionControlUsername",
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: "The username associated with the platform hub version control settings",
		Default:     strutil.StrPointer(resource.Credentials.Username),
	}

	block = gohcl.EncodeAsBlock(usernameVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	secretVariableResource := terraform.TerraformVariable{
		Name:        "PlatformHubVersionControlPassword",
		Type:        "string",
		Nullable:    false,
		Sensitive:   true,
		Description: "The secret variable value associated with the platform hub version control settings",
	}

	if c.DummySecretVariableValues {
		secretVariableResource.Default = c.DummySecretGenerator.GetDummySecret()
		dependencies.AddDummy(data.DummyVariableReference{
			VariableName: "PlatformHubVersionControlPassword",
			ResourceName: "PlatformHubVersionControlPassword",
			ResourceType: "PlatformHubVersionControl",
		})
	}

	block = gohcl.EncodeAsBlock(secretVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)

	return string(file.Bytes()), nil
}
