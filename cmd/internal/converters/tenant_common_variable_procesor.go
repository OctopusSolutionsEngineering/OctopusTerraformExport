package converters

import (
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/args"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"strings"
)

// TenantCommonVariableProcessor is used to serialize the tenant common variables.
// Tenant common variables are the one resource that is neither space level nor project level. Instead,
// these variables can only be defined when a tenant is linked to a project that is linked to a library variable set.
// This breaks the pattern of the other resources which are either space level or project level.
// So this processor can either create common variables when the entire space is serialized, or projects can
// export tenant common variables as part of the project.
type TenantCommonVariableProcessor struct {
	Excluder                     ExcludeByName
	ExcludeAllProjects           bool
	ExcludeAllTenantVariables    bool
	ExcludeTenantVariables       args.StringSliceArgs
	ExcludeTenantVariablesExcept args.StringSliceArgs
	ExcludeTenantVariablesRegex  args.StringSliceArgs
}

func (c TenantCommonVariableProcessor) ConvertTenantCommonVariable(stateless bool, tenantVariable octopus.TenantVariable, tenantVariableId string, tenantVariableValue any, libraryVariableSet octopus.LibraryVariableSet, commonVariableIndex int, dependencies *data.ResourceDetailsCollection) error {
	// A tenant common variable needs the tenant to be linked to a project that then links to the library
	// variable set that defines the common variable. If we are excluding all projects, there is no way
	// to define any common variables.
	if c.ExcludeAllProjects {
		return nil
	}

	libraryVariableSetVariableName := lo.Filter(libraryVariableSet.Templates, func(item octopus.Template, index int) bool {
		return item.Id == tenantVariableId
	})

	if len(libraryVariableSetVariableName) != 0 {
		// Do not export excluded variables
		if c.Excluder.IsResourceExcludedWithRegex(strutil.EmptyIfNil(libraryVariableSetVariableName[0].Name),
			c.ExcludeAllTenantVariables,
			c.ExcludeTenantVariables,
			c.ExcludeTenantVariablesRegex,
			c.ExcludeTenantVariablesExcept) {
			return nil
		}
	}

	var count *string = nil
	if stateless {
		count = strutil.StrPointer("${length(data." + octopusdeployTenantsDataType + ".tenant_" +
			sanitizer.SanitizeName(tenantVariable.TenantName) + ".tenants) != 0 ? 0 : 1}")
	}

	variableName := "tenantcommonvariable" + fmt.Sprint(commonVariableIndex) + "_" + sanitizer.SanitizeName(tenantVariable.TenantName)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + variableName + ".tf"
	thisResource.Id = tenantVariableId
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_tenant_common_variable." + variableName + ".id}"

	/*
		Tenants can define secrets, in which case value is an object indicating the state of the
		secret, but not the value. In this case we can only export an empty string.
		TODO: Create a variable to override this value if needed.
	*/
	fixedValue := ""
	if stringValue, ok := tenantVariableValue.(string); ok {
		fixedValue = stringValue
	}

	thisResource.ToHcl = func() (string, error) {
		file := hclwrite.NewEmptyFile()
		terraformResource := terraform.TerraformTenantCommonVariable{
			Type:                 "octopusdeploy_tenant_common_variable",
			Name:                 variableName,
			Count:                count,
			Id:                   nil,
			LibraryVariableSetId: dependencies.GetResource("LibraryVariableSets", libraryVariableSet.Id),
			TemplateId:           dependencies.GetResource("CommonTemplateMap", tenantVariableId),
			TenantId:             dependencies.GetResource("Tenants", tenantVariable.TenantId),
			Value:                &fixedValue,
		}
		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		// common variables rely on the link between a tenant and a project, and this can only
		// be expressed in a depends_on attribute. We rely on the fact that the ID of the tenant project
		// links has the tenant ID as a prefix.
		tenantProjects := lo.FilterMap(dependencies.GetAllResource("TenantProject"), func(item data.ResourceDetails, index int) (string, bool) {
			return hcl.RemoveInterpolation(item.Dependency), strings.HasPrefix(item.Id, tenantVariable.TenantId)
		})
		hcl.WriteUnquotedAttribute(block, "depends_on", "["+strings.Join(tenantProjects[:], ",")+"]")

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)

	return nil
}

func (c TenantCommonVariableProcessor) GetResourceType() string {
	return "TenantVariables/All"
}
