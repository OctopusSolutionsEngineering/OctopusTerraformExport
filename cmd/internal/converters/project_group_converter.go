package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
)

type ProjectGroupConverter struct {
	Client client.OctopusClient
}

func (c ProjectGroupConverter) ToHcl(dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.ProjectGroup]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Project Group: " + resource.Id)
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

	resource := octopus.ProjectGroup{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Project Group: " + resource.Id)
	return c.toHcl(resource, false, false, dependencies)
}

func (c ProjectGroupConverter) ToHclLookupById(id string, dependencies *ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.ProjectGroup{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	return c.toHcl(resource, false, true, dependencies)
}

func (c ProjectGroupConverter) toHcl(resource octopus.ProjectGroup, recursive bool, lookup bool, dependencies *ResourceDetailsCollection) error {
	thisResource := ResourceDetails{}

	forceLookup := lookup || resource.Name == "Default Project Group"

	projectName := "project_group_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/projectgroup_" + projectName + ".tf"
	thisResource.Id = resource.Id
	thisResource.ResourceType = c.GetResourceType()

	if forceLookup {
		thisResource.Lookup = "${data.octopusdeploy_project_groups." + projectName + ".project_groups[0].id}"
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform.TerraformProjectGroupData{
				Type:        "octopusdeploy_project_groups",
				Name:        projectName,
				Ids:         nil,
				PartialName: "${var." + projectName + "_name}",
				Skip:        0,
				Take:        1,
			}
			file := hclwrite.NewEmptyFile()
			c.writeProjectNameVariable(file, projectName, resource.Name)
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a project group called ${var."+projectName+"_name}. This resource must exist in the space before this Terraform configuration is applied.", "length(self.project_groups) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	} else {
		thisResource.Lookup = "${octopusdeploy_project_group." + projectName + ".id}"
		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform.TerraformProjectGroup{
				Type:         "octopusdeploy_project_group",
				Name:         projectName,
				ResourceName: "${var." + projectName + "_name}",
				Description:  resource.Description,
			}
			file := hclwrite.NewEmptyFile()

			c.writeProjectNameVariable(file, projectName, resource.Name)

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

func (c ProjectGroupConverter) writeProjectNameVariable(file *hclwrite.File, projectName string, projectGroupResourceName string) {
	projectNameVariableResource := terraform.TerraformVariable{
		Name:        projectName + "_name",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The name of the project group to lookup",
		Default:     &projectGroupResourceName,
	}

	block := gohcl.EncodeAsBlock(projectNameVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)
}

func (c ProjectGroupConverter) GetResourceType() string {
	return "ProjectGroups"
}
