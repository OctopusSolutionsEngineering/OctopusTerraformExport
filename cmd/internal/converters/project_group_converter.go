package converters

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	octopus2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	terraform2 "github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
)

type ProjectGroupConverter struct {
	Client client.OctopusClient
}

func (c ProjectGroupConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus2.GeneralCollection[octopus2.ProjectGroup]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		err = c.toHcl(resource, false, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c ProjectGroupConverter) ToHclById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.ProjectGroup{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, false, false, dependencies)
}

func (c ProjectGroupConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus2.ProjectGroup{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, false, true, dependencies)
}

func (c ProjectGroupConverter) toHcl(resource octopus2.ProjectGroup, recursive bool, lookup bool, dependencies *ResourceDetailsCollection) error {
	thisResource := ResourceDetails{}

	forceLookup := lookup || resource.Name == "Default Project Group"

	projectName := "project_group_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/projectgroup_" + projectName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()

	if forceLookup {
		thisResource.Lookup = "${data.octopusdeploy_project_groups." + projectName + ".project_groups[0].id}"
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform2.TerraformProjectGroupData{
				Type:        "octopusdeploy_project_groups",
				Name:        projectName,
				Ids:         nil,
				PartialName: resource.Name,
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "data"))

			return string(file.Bytes()), nil
		}
	} else {
		thisResource.Lookup = "${octopusdeploy_project_group." + projectName + ".id}"
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform2.TerraformProjectGroup{
				Type:         "octopusdeploy_project_group",
				Name:         projectName,
				ResourceName: resource.Name,
				Description:  resource.Description,
			}
			file := hclwrite.NewEmptyFile()

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
				Type: hclsyntax.TokenComment,
				Bytes: []byte("# Import existing resources with the following commands:\n" +
					"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + resource.Name + "\") | .Id')\n" +
					"# terraform import octopusdeploy_project_group." + projectName + " ${RESOURCE_ID}\n"),
				SpacesBefore: 0,
			}})

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

func (c ProjectGroupConverter) GetResourceType() string {
	return "ProjectGroups"
}
