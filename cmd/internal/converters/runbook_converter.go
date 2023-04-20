package converters

import (
	"fmt"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/client"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/mcasperson/OctopusTerraformExport/cmd/internal/sanitizer"
)

type RunbookConverter struct {
	Client                  client.OctopusClient
	RunbookProcessConverter ConverterAndLookupByIdAndName
}

func (c RunbookConverter) ToHclByIdAndName(projectId string, projectName string, dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Runbook]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return err
	}

	for _, runbook := range collection.Items {
		err = c.toHcl(runbook, projectName, true, false, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c RunbookConverter) ToHclLookupByIdAndName(projectId string, projectName string, dependencies *ResourceDetailsCollection) error {
	collection := octopus.GeneralCollection[octopus.Runbook]{}
	err := c.Client.GetAllResources(c.GetGroupResourceType(projectId), &collection)

	if err != nil {
		return err
	}

	for _, runbook := range collection.Items {
		err = c.toHcl(runbook, projectName, false, true, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c RunbookConverter) toHcl(runbook octopus.Runbook, projectName string, recursive bool, lookups bool, dependencies *ResourceDetailsCollection) error {
	thisResource := ResourceDetails{}

	resourceNameSuffix := sanitizer.SanitizeName(projectName) + "_" + sanitizer.SanitizeName(runbook.Name)
	runbookName := "runbook_" + resourceNameSuffix

	err := c.exportChildDependencies(recursive, lookups, runbook, resourceNameSuffix, dependencies)

	if err != nil {
		return err
	}

	thisResource.FileName = "space_population/" + runbookName + ".tf"
	thisResource.Id = runbook.Id
	thisResource.ResourceType = c.GetResourceType()
	thisResource.Lookup = "${octopusdeploy_runbook." + runbookName + ".id}"
	thisResource.ToHcl = func() (string, error) {

		terraformResource := terraform.TerraformRunbook{
			Type:                     "octopusdeploy_runbook",
			Name:                     runbookName,
			ResourceName:             "${var." + runbookName + "_name}",
			ProjectId:                dependencies.GetResource("Projects", runbook.ProjectId),
			EnvironmentScope:         runbook.EnvironmentScope,
			Environments:             dependencies.GetResources("Environments", runbook.Environments...),
			ForcePackageDownload:     runbook.ForcePackageDownload,
			DefaultGuidedFailureMode: runbook.DefaultGuidedFailureMode,
			Description:              runbook.Description,
			MultiTenancyMode:         runbook.MultiTenancyMode,
			RetentionPolicy:          c.convertRetentionPolicy(runbook),
			ConnectivityPolicy:       c.convertConnectivityPolicy(runbook),
		}
		file := hclwrite.NewEmptyFile()

		c.writeProjectNameVariable(file, runbookName, runbook.Name)

		// Add a comment with the import command
		baseUrl, _ := c.Client.GetSpaceBaseUrl()
		file.Body().AppendUnstructuredTokens([]*hclwrite.Token{{
			Type: hclsyntax.TokenComment,
			Bytes: []byte("# Import existing resources with the following commands:\n" +
				"# RESOURCE_ID=$(curl -H \"X-Octopus-ApiKey: ${OCTOPUS_CLI_API_KEY}\" " + baseUrl + "/" + c.GetResourceType() + " | jq -r '.Items[] | select(.Name==\"" + runbook.Name + "\") | .Id')\n" +
				"# terraform import octopusdeploy_runbook." + runbookName + " ${RESOURCE_ID}\n"),
			SpacesBefore: 0,
		}})

		file.Body().AppendBlock(gohcl.EncodeAsBlock(terraformResource, "resource"))

		return string(file.Bytes()), nil
	}
	dependencies.AddResource(thisResource)

	return nil
}

func (c RunbookConverter) GetResourceType() string {
	return "Runbooks"
}

func (c RunbookConverter) GetGroupResourceType(projectId string) string {
	return "Projects/" + projectId + "/runbooks"
}

func (c RunbookConverter) writeProjectNameVariable(file *hclwrite.File, projectName string, projectResourceName string) {
	secretVariableResource := terraform.TerraformVariable{
		Name:        projectName + "_name",
		Type:        "string",
		Nullable:    false,
		Sensitive:   false,
		Description: "The name of the project exported from " + projectResourceName,
		Default:     &projectResourceName,
	}

	block := gohcl.EncodeAsBlock(secretVariableResource, "variable")
	hcl.WriteUnquotedAttribute(block, "type", "string")
	file.Body().AppendBlock(block)
}

func (c RunbookConverter) convertTemplates(actionPackages []octopus.Template, projectName string) ([]terraform.TerraformTemplate, []ResourceDetails) {
	templateMap := make([]ResourceDetails, 0)
	collection := make([]terraform.TerraformTemplate, 0)
	for i, v := range actionPackages {
		collection = append(collection, terraform.TerraformTemplate{
			Name:            v.Name,
			Label:           v.Label,
			HelpText:        v.HelpText,
			DefaultValue:    v.DefaultValue,
			DisplaySettings: v.DisplaySettings,
		})

		templateMap = append(templateMap, ResourceDetails{
			Id:           v.Id,
			ResourceType: "ProjectTemplates",
			Lookup:       "${octopusdeploy_project." + projectName + ".template[" + fmt.Sprint(i) + "].id}",
			FileName:     "",
			ToHcl:        nil,
		})
	}
	return collection, templateMap
}

func (c RunbookConverter) convertConnectivityPolicy(runbook octopus.Runbook) *terraform.TerraformConnectivityPolicy {
	return &terraform.TerraformConnectivityPolicy{
		AllowDeploymentsToNoTargets: runbook.ConnectivityPolicy.AllowDeploymentsToNoTargets,
		ExcludeUnhealthyTargets:     runbook.ConnectivityPolicy.ExcludeUnhealthyTargets,
		SkipMachineBehavior:         runbook.ConnectivityPolicy.SkipMachineBehavior,
	}
}

func (c RunbookConverter) convertRetentionPolicy(runbook octopus.Runbook) *terraform.RetentionPolicy {
	return &terraform.RetentionPolicy{
		QuantityToKeep:    runbook.RunRetentionPolicy.QuantityToKeep,
		ShouldKeepForever: runbook.RunRetentionPolicy.ShouldKeepForever,
	}
}

func (c RunbookConverter) exportChildDependencies(recursive bool, lookup bool, runbook octopus.Runbook, runbookName string, dependencies *ResourceDetailsCollection) error {
	// Export the deployment process
	if runbook.RunbookProcessId != nil {
		var err error
		if lookup {
			err = c.RunbookProcessConverter.ToHclLookupByIdAndName(*runbook.RunbookProcessId, runbookName, dependencies)
		} else {
			err = c.RunbookProcessConverter.ToHclByIdAndName(*runbook.RunbookProcessId, runbookName, dependencies)
		}

		if err != nil {
			return err
		}
	}

	return nil
}
