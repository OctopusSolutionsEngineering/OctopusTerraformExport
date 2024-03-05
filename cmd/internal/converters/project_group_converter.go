package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const octopusdeployProjectGroupsDataType = "octopusdeploy_project_groups"
const octopusdeployProjectGroupResourceType = "octopusdeploy_project_group"

type ProjectGroupConverter struct {
	Client   client.OctopusClient
	ErrGroup *errgroup.Group
}

func (c ProjectGroupConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c ProjectGroupConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c ProjectGroupConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.ProjectGroup]{}
	err := c.Client.GetAllResources(c.GetResourceType(), &collection)

	if err != nil {
		return err
	}

	for _, resource := range collection.Items {
		zap.L().Info("Project Group: " + resource.Id)
		err = c.toHcl(resource, false, false, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c ProjectGroupConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c ProjectGroupConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c ProjectGroupConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {

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
	return c.toHcl(resource, false, false, stateless, dependencies)
}

func (c ProjectGroupConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
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

	return c.toHcl(resource, false, true, false, dependencies)
}

func (c ProjectGroupConverter) buildData(resourceName string, name string) terraform.TerraformProjectGroupData {
	return terraform.TerraformProjectGroupData{
		Type:        octopusdeployProjectGroupsDataType,
		Name:        name,
		Ids:         nil,
		PartialName: resourceName,
		Skip:        0,
		Take:        1,
	}
}

// writeData appends the data block for stateless modules
func (c ProjectGroupConverter) writeData(file *hclwrite.File, name string, resourceName string) {
	terraformResource := c.buildData(resourceName, name)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c ProjectGroupConverter) toHcl(resource octopus.ProjectGroup, recursive bool, lookup bool, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	thisResource := data.ResourceDetails{}

	forceLookup := lookup || resource.Name == "Default Project Group"

	projectName := "project_group_" + sanitizer.SanitizeName(resource.Name)

	thisResource.FileName = "space_population/projectgroup_" + projectName + ".tf"
	thisResource.Id = resource.Id
	thisResource.Name = resource.Name
	thisResource.ResourceType = c.GetResourceType()

	if forceLookup {
		thisResource.Lookup = "${data." + octopusdeployProjectGroupsDataType + "." + projectName + ".project_groups[0].id}"
		thisResource.ToHcl = func() (string, error) {
			terraformResource := c.buildData("${var."+projectName+"_name}", projectName)
			file := hclwrite.NewEmptyFile()
			c.writeProjectNameVariable(file, projectName, resource.Name)
			block := gohcl.EncodeAsBlock(terraformResource, "data")
			hcl.WriteLifecyclePostCondition(block, "Failed to resolve a project group called ${var."+projectName+"_name}. This resource must exist in the space before this Terraform configuration is applied.", "length(self.project_groups) != 0")
			file.Body().AppendBlock(block)

			return string(file.Bytes()), nil
		}
	} else {
		if stateless {
			thisResource.Lookup = "${length(data." + octopusdeployProjectGroupsDataType + "." + projectName + ".project_groups) != 0 " +
				"? data." + octopusdeployProjectGroupsDataType + "." + projectName + ".project_groups[0].id " +
				": " + octopusdeployProjectGroupResourceType + "." + projectName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployProjectGroupResourceType + "." + projectName + "}"
		} else {
			thisResource.Lookup = "${" + octopusdeployProjectGroupResourceType + "." + projectName + ".id}"
		}

		thisResource.ToHcl = func() (string, error) {
			terraformResource := terraform.TerraformProjectGroup{
				Type:         octopusdeployProjectGroupResourceType,
				Name:         projectName,
				ResourceName: "${var." + projectName + "_name}",
				Description:  resource.Description,
			}
			file := hclwrite.NewEmptyFile()

			if stateless {
				c.writeData(file, projectName, "${var."+projectName+"_name}")
				terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeployProjectGroupsDataType + "." + projectName + ".project_groups) != 0 ? 0 : 1}")
			}

			c.writeProjectNameVariable(file, projectName, resource.Name)

			// Add a comment with the import command
			baseUrl, _ := c.Client.GetSpaceBaseUrl()
			file.Body().AppendUnstructuredTokens(hcl.WriteImportComments(baseUrl, c.GetResourceType(), resource.Name, octopusdeployProjectGroupResourceType, projectName))

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			if stateless {
				hcl.WriteLifecyclePreventDestroyAttribute(block)
			}

			file.Body().AppendBlock(block)

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
