package singleconverter

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/internal/util"
)

type SingleProjectGroupConverter struct {
	Client client.OctopusClient
}

func (c SingleProjectGroupConverter) ToHclById(id string, recursive bool, dependencies *ResourceDetailsCollection) error {
	resource := octopus.ProjectGroup{}
	err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	thisResource := ResourceDetails{}

	projectName := "project_group_" + util.SanitizeNamePointer(resource.Name)

	thisResource.FileName = "space_population/projectgroup_" + projectName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_project_group." + projectName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		if *resource.Name == "Default Project Group" {
			// todo - create lookup for existing project group
			return "", nil
		} else {
			terraformResource := terraform.TerraformProjectGroup{
				Type:         "octopusdeploy_project_group",
				Name:         projectName,
				ResourceName: resource.Name,
				Description:  resource.Description,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

			return string(file.Bytes()), nil
		}
	}

	if recursive {
		// export child projects
	}

	dependencies.AddResource(thisResource)
	return nil
}

func (c SingleProjectGroupConverter) GetResourceType() string {
	return "ProjectGroups"
}
