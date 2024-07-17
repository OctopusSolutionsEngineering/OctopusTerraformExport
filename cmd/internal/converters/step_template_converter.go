package converters

import (
	"encoding/json"
	"fmt"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/steptemplate"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const octopusdeployStepTemplateResourceType = "shell_script"
const octopusdeployStepTemplateDataType = "external"

// StepTemplateConverter is a placeholder for real step templates. We use the shell_script resource type to run custom
// PowerShell scripts to manage step templates, and the external data source type to query the Octopus API.
// This implementation will eventually be replaced when step templates are fully supported by the Octopus Terraform provider.
type StepTemplateConverter struct {
	ErrGroup                   *errgroup.Group
	Client                     client.OctopusClient
	ExcludeAllStepTemplates    bool
	ExcludeStepTemplates       []string
	ExcludeStepTemplatesRegex  []string
	ExcludeStepTemplatesExcept []string
	Excluder                   ExcludeByName
	LimitResourceCount         int
	GenerateImportScripts      bool
}

func (c StepTemplateConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c StepTemplateConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {

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
	return "ActionTemplates"
}

func (c StepTemplateConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllStepTemplates {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[steptemplate.StepTemplate]{
		Client: c.Client,
	}

	done := make(chan struct{})
	defer close(done)

	channel := batchClient.GetAllResourcesBatch(done, c.GetResourceType())

	for resourceWrapper := range channel {
		if resourceWrapper.Err != nil {
			return resourceWrapper.Err
		}

		resource := resourceWrapper.Res

		zap.L().Info("Step Template: " + resource.Id)
		err := c.toHcl(resource, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c StepTemplateConverter) toHcl(template steptemplate.StepTemplate, stateless bool, dependencies *data.ResourceDetailsCollection) error {
	// Ignore excluded step templates
	if c.Excluder.IsResourceExcludedWithRegex(template.Name, c.ExcludeAllStepTemplates, c.ExcludeStepTemplates, c.ExcludeStepTemplatesRegex, c.ExcludeStepTemplatesExcept) {
		return nil
	}

	if c.LimitResourceCount > 0 && len(dependencies.GetAllResource(c.GetResourceType())) >= c.LimitResourceCount {
		zap.L().Info(c.GetResourceType() + " hit limit of " + fmt.Sprint(c.LimitResourceCount) + " - skipping " + template.Id)
		return nil
	}

	stepTemplateName := "steptemplate_" + sanitizer.SanitizeName(template.Name)

	/*if c.GenerateImportScripts {
		c.toBashImport(stepTemplateName, target.Name, dependencies)
		c.toPowershellImport(stepTemplateName, target.Name, dependencies)
	}*/

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + stepTemplateName + ".tf"
	thisResource.Id = template.Id
	thisResource.Name = template.Name
	thisResource.ResourceType = c.GetResourceType()

	if stateless {
		thisResource.Lookup = "${length(keys(data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ")) != 0 " +
			"? values(data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ")[0] " +
			": " + octopusdeployStepTemplateResourceType + "." + stepTemplateName + "[0].id}"
		thisResource.Dependency = "${" + octopusdeployStepTemplateResourceType + "." + stepTemplateName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployStepTemplateResourceType + "." + stepTemplateName + ".id}"
	}

	thisResource.ToHcl = func() (string, error) {

		stepTemplateJson, err := json.Marshal(template)

		if err != nil {
			return "", err
		}

		terraformResource := terraform.TerraformShellScript{
			Type: octopusdeployStepTemplateResourceType,
			Name: stepTemplateName,
			LifecycleCommands: terraform.TerraformShellScriptLifecycleCommands{
				Read: strutil.StrPointer(strutil.StripMultilineWhitespace(`$state = Read-Host | ConvertFrom-JSON
					if ([string]::IsNullOrEmpty($state.Id)) {
						Write-Host "{}"
					} else {
						$headers = @{ "X-Octopus-ApiKey" = $env:APIKEY }
						$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/$($env:SPACEID)/actiontemplates/$($state.Id)" -Method GET -Headers $headers
						Write-Host $response.content
					}`)),
				Create: strutil.StripMultilineWhitespace("$json = " + strutil.PowershellEscape(string(stepTemplateJson)) + "\n" +
					`$headers = @{ "X-Octopus-ApiKey" = $env:APIKEY }
					$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/$($env:SPACEID)/actiontemplates" -ContentType "application/json" -Method POST -Body $json -Headers $headers
					Write-Host $response.content`),
				Update: strutil.StrPointer(strutil.StripMultilineWhitespace("$json = " + strutil.PowershellEscape(string(stepTemplateJson)) + "\n" +
					`$state = Read-Host | ConvertFrom-JSON
					if ([string]::IsNullOrEmpty($state.Id)) {
						Write-Host "{}"
					} else {
						$headers = @{ "X-Octopus-ApiKey" = $env:APIKEY }
						$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/$($env:SPACEID)/actiontemplates/$($state.Id)" -ContentType "application/json" -Method PUT -Body $json -Headers $headers
						Write-Host $response.content
					}`)),
				Delete: strutil.StripMultilineWhitespace(`$state = Read-Host | ConvertFrom-JSON
					if (-not [string]::IsNullOrEmpty($state.Id)) {
						$headers = @{ "X-Octopus-ApiKey" = $env:APIKEY }
						$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/$($env:SPACEID)/actiontemplates/$($state.Id)" -Method DELETE -Headers $headers
					}`),
			},
			Environment: map[string]string{
				"SERVER":  "${var.octopus_server}",
				"SPACEID": "${var.octopus_space_id}",
			},
			SensitiveEnvironment: map[string]string{
				"APIKEY": "${var.octopus_apikey}",
			},
			WorkingDirectory: strutil.StrPointer("${path.module}"),
		}
		file := hclwrite.NewEmptyFile()

		if stateless {
			c.writeData(file, template, stepTemplateName)
			terraformResource.Count = strutil.StrPointer("${length(data." + octopusdeploySshConnectionDeploymentTargetDataType + "." + stepTemplateName + ".deployment_targets) != 0 ? 0 : 1}")
		}

		block := gohcl.EncodeAsBlock(terraformResource, "resource")

		if stateless {
			hcl.WriteLifecyclePreventDestroyAttribute(block)
		}

		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

// writeData appends the data block for stateless modules
func (c StepTemplateConverter) writeData(file *hclwrite.File, resource steptemplate.StepTemplate, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c StepTemplateConverter) buildData(resourceName string, resource steptemplate.StepTemplate) terraform.TerraformExternalData {
	return terraform.TerraformExternalData{
		Type: octopusdeployStepTemplateDataType,
		Name: resourceName,
		Program: []string{
			"pwsh",
			"-Command",
			`$query = Read-Host | ConvertFrom-JSON
			$headers = @{ "X-Octopus-ApiKey" = $query.apikey }
			$response = Invoke-WebRequest -Uri "$($query.server)/api/$($query.spaceid)/actiontemplates" -Method GET -Headers $headers
			$keyValueResponse = @{}
			$response.content | ConvertFrom-JSON | Select-Object -Expand Items | ? {$_.Name -eq $query.name} | % {$keyValueResponse[$_.Id] = $_.Name}
			Write-Host ($keyValueResponse | ConvertTo-JSON)`},
		Query: map[string]string{
			"name":    resource.Name,
			"server":  "${var.octopus_server}",
			"apikey":  "${var.octopus_apikey}",
			"spaceid": "${var.octopus_space_id}",
		},
	}
}
