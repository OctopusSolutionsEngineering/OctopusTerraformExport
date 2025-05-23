package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/naming"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"strings"
)

type TenantProjectVariableConverter struct {
	Excluder                     ExcludeByName
	ExcludeAllProjects           bool
	ExcludeAllTenantVariables    bool
	ExcludeTenantVariables       args.StringSliceArgs
	ExcludeTenantVariablesExcept args.StringSliceArgs
	ExcludeTenantVariablesRegex  args.StringSliceArgs
	DummySecretVariableValues    bool
	DummySecretGenerator         dummy.DummySecretGenerator
}

func (c TenantProjectVariableConverter) ConvertTenantProjectVariable(stateless bool, tenantVariable octopus.TenantVariable, projectVariable octopus.ProjectVariable, environmentId string, value any, projectVariableIndex int, templateId string, dependencies *data.ResourceDetailsCollection) error {
	variableName := "tenantprojectvariable_" + fmt.Sprint(projectVariableIndex) + "_" + sanitizer.SanitizeName(tenantVariable.TenantName)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + variableName + ".tf"
	thisResource.Id = templateId
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${" + octopusdeployTenantProjectVariableResourceType + "." + variableName + ".id}"

	// Assume the tenant has added the data block to resolve existing tenants. Use that data block
	// to test if any of the tenant variables should be created.
	var count *string = nil
	if stateless {
		count = strutil.StrPointer("${length(data." + octopusdeployTenantsDataType + ".tenant_" +
			sanitizer.SanitizeName(tenantVariable.TenantName) + ".tenants) != 0 ? 0 : 1}")
	}

	if stateless {
		tenantName := "tenant_" + sanitizer.SanitizeName(tenantVariable.TenantName)
		thisResource.Lookup = "${length(data." + octopusdeployTenantsDataType + "." + tenantName + ".tenants) != 0 " +
			"? '' " +
			": " + octopusdeployTenantProjectVariableResourceType + "." + variableName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployTenantProjectVariableResourceType + "." + variableName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployTenantProjectVariableResourceType + "." + variableName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {
		file := hclwrite.NewEmptyFile()

		tenantProjectVariableValue := terraform.TerraformVariable{
			Name:        naming.TenantVariableValueName(tenantVariable),
			Type:        "string",
			Nullable:    false,
			Sensitive:   false,
			Description: "The value of the tenant project variable",
			Default:     nil,
		}

		// Define a secret value with an optional dummy default
		if _, ok := value.(map[string]any); ok {
			tenantProjectVariableValue.Name = naming.TenantVariableSecretName(tenantVariable)

			if c.DummySecretVariableValues {
				tenantProjectVariableValue.Default = c.DummySecretGenerator.GetDummySecret()
			}
			dependencies.AddDummy(data.DummyVariableReference{
				VariableName: tenantProjectVariableValue.Name,
				ResourceName: tenantVariable.TenantName,
				ResourceType: c.GetResourceType(),
			})
		}

		// Define the default value
		if stringValue, ok := value.(string); ok {
			tenantProjectVariableValue.Default = strutil.StrPointer(strutil.EscapeDollarCurly(stringValue))
		}

		variableBlock := gohcl.EncodeAsBlock(tenantProjectVariableValue, "variable")
		hcl.WriteUnquotedAttribute(variableBlock, "type", "string")
		file.Body().AppendBlock(variableBlock)

		terraformResource := terraform.TerraformTenantProjectVariable{
			Type:          octopusdeployTenantProjectVariableResourceType,
			Name:          variableName,
			Count:         count,
			Id:            nil,
			EnvironmentId: dependencies.GetResource("Environments", environmentId),
			ProjectId:     dependencies.GetResource("Projects", projectVariable.ProjectId),
			TemplateId:    dependencies.GetResource("ProjectTemplates", templateId),
			TenantId:      dependencies.GetResource("Tenants", tenantVariable.TenantId),
			Value:         strutil.StrPointer("${var." + tenantProjectVariableValue.Name + "}"),
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		// common variables rely on the link between a tenant and a project, and this can only
		// be expressed in a depends_on attribute. We rely on the fact that the ID of the tenant project
		// links has the tenant ID as a prefix.
		tenantProjects := lo.FilterMap(dependencies.GetAllResource("TenantProject"), func(item data.ResourceDetails, index int) (string, bool) {
			return hcl.RemoveInterpolation(item.Dependency), strings.HasPrefix(item.Id, tenantVariable.TenantId)
		})
		hcl.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(tenantProjects[:], ",")+"]")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)
		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)

	return nil
}

func (c TenantProjectVariableConverter) GetResourceType() string {
	return "TenantVariables/All"
}
