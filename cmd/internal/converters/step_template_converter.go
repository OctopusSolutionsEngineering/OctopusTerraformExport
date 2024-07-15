package converters

import (
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
)

// StepTemplateConverter is a placeholder for real step templates. The exported resource is a Terraform variable,
// which allows a quick way to remap a step template ID as there is one place the ID needs to be updated.
// In future there should be a real step template resource, but this will do for now.
type StepTemplateConverter struct {
}

func (c StepTemplateConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {

	stepTemplateName := sanitizer.SanitizeName(id)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + stepTemplateName + ".tf"
	thisResource.Id = id
	thisResource.Name = id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${var." + stepTemplateName + "}"
	thisResource.ToHcl = func() (string, error) {

		variable := terraform.TerraformVariable{
			Name:        stepTemplateName,
			Type:        "string",
			Nullable:    false,
			Sensitive:   false,
			Description: "Step template ID",
			Default:     &id,
		}

		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(variable, "variable")
		hcl.WriteUnquotedAttribute(block, "type", "string")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c StepTemplateConverter) GetResourceType() string {
	return "StepTemplates"
}
