package converters

import (
	"encoding/json"
	"fmt"
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
	"strconv"
)

const octopusdeployStepTemplateResourceType = "shell_script"
const octopusdeployStepTemplateDataType = "external"

// StepTemplateConverter is a placeholder for real step templates. We use the shell_script resource type to run custom
// PowerShell scripts to manage step templates, and the external data source type to query the Octopus API.
// This implementation will eventually be replaced when step templates are fully supported by the Octopus Terraform provider.
type StepTemplateConverter struct {
	ErrGroup                        *errgroup.Group
	Client                          client.OctopusClient
	ExcludeAllStepTemplates         bool
	ExcludeStepTemplates            []string
	ExcludeStepTemplatesRegex       []string
	ExcludeStepTemplatesExcept      []string
	Excluder                        ExcludeByName
	LimitResourceCount              int
	GenerateImportScripts           bool
	ExperimentalEnableStepTemplates bool
}

func (c StepTemplateConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	if !c.ExperimentalEnableStepTemplates {
		return
	}

	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c StepTemplateConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	if !c.ExperimentalEnableStepTemplates {
		return
	}
}

func (c StepTemplateConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	if !c.ExperimentalEnableStepTemplates {
		return nil
	}

	if c.ExcludeAllStepTemplates {
		return nil
	}

	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	resource := octopus.StepTemplate{}
	_, err := c.Client.GetResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return err
	}

	zap.L().Info("Step Template: " + resource.Id)
	return c.toHcl(resource, false, dependencies)
}

func (c StepTemplateConverter) GetResourceType() string {
	return "ActionTemplates"
}

func (c StepTemplateConverter) allToHcl(stateless bool, dependencies *data.ResourceDetailsCollection) error {
	if c.ExcludeAllStepTemplates {
		return nil
	}

	batchClient := client.BatchingOctopusApiClient[octopus.StepTemplate]{
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

func (c StepTemplateConverter) toHcl(template octopus.StepTemplate, stateless bool, dependencies *data.ResourceDetailsCollection) error {
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

	stepTemplateResource := data.ResourceDetails{}
	stepTemplateResource.FileName = "space_population/" + stepTemplateName + ".json"
	stepTemplateResource.Id = template.Id
	stepTemplateResource.Name = template.Name
	stepTemplateResource.ResourceType = "ActionTemplatesJson"
	stepTemplateResource.ToHcl = func() (string, error) {
		// Remove the version from the template before marshalling
		template.Version = nil
		stepTemplateJson, err := json.Marshal(template)
		return string(stepTemplateJson), err
	}
	dependencies.AddResource(stepTemplateResource)

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + stepTemplateName + ".tf"
	thisResource.Id = template.Id
	thisResource.Name = template.Name
	thisResource.VersionLookup = "${" + octopusdeployStepTemplateResourceType + "." + stepTemplateName + ".output.Version}"
	thisResource.VersionCurrent = strconv.Itoa(*template.Version)
	thisResource.ResourceType = c.GetResourceType()

	if stateless {
		thisResource.Lookup = "${length(keys(data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ")) != 0 " +
			"? values(data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ")[0] " +
			": " + octopusdeployStepTemplateResourceType + "." + stepTemplateName + "[0].output.Id}"
		thisResource.Dependency = "${" + octopusdeployStepTemplateResourceType + "." + stepTemplateName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployStepTemplateResourceType + "." + stepTemplateName + ".output.Id}"
	}

	thisResource.ToHcl = func() (string, error) {

		environmentVars := map[string]string{}
		environmentVars["VERSION"] = thisResource.VersionCurrent
		for _, v2 := range dependencies.GetAllResource("Feeds") {
			environmentVars["FEED_"+v2.Id] = v2.Lookup
		}

		terraformResource := terraform.TerraformShellScript{
			Type: octopusdeployStepTemplateResourceType,
			Name: stepTemplateName,
			LifecycleCommands: terraform.TerraformShellScriptLifecycleCommands{
				Read: strutil.StrPointer(strutil.StripMultilineWhitespace(`$host.ui.WriteErrorLine('Reading step template')
					$state = Read-Host | ConvertFrom-JSON
					if ([string]::IsNullOrEmpty($state.Id)) {
						$host.ui.WriteErrorLine('State ID is empty')
						Write-Host "{}"
					} else {
						$host.ui.WriteErrorLine('State ID is ($state.Id)')
						$headers = @{ "X-Octopus-ApiKey" = $env:APIKEY }
						$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/$($env:SPACEID)/actiontemplates/$($state.Id)" -Method GET -Headers $headers
						# Strip out the last modified by details
						$stepTemplateObject = $response.content | ConvertFrom-Json
						$stepTemplateObject.PSObject.Properties.Remove('LastModifiedBy')
						$stepTemplateObject.PSObject.Properties.Remove('LastModifiedOn')
						Write-Host $($stepTemplateObject | ConvertTo-Json -Depth 100)
					}`)),
				Create: strutil.StripMultilineWhitespace(`$host.ui.WriteErrorLine('Create step template')
					$headers = @{ "X-Octopus-ApiKey" = $env:APIKEY }
					$body = Get-Content -Raw -Path ` + stepTemplateName + `.json

					# Replace feed IDs with lookup values passed in via env vars
					gci env:* | ? {$_.Name -like "FEED_*} | % {$body = $body.Replace($_.Name.Replace("FEED_", ""), $_.Value)}

					$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/$($env:SPACEID)/actiontemplates" -ContentType "application/json" -Method POST -Body $body -Headers $headers
					$stepTemplate = $response.content | ConvertFrom-Json
					# Import any new step template twice to ensure the version of a new template is at least 1.
					$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/$($env:SPACEID)/actiontemplates/$($stepTemplate.Id)" -ContentType "application/json" -Method PUT -Body $body -Headers $headers
					# Strip out the last modified by details
					$stepTemplateObject = $response.content | ConvertFrom-Json
					$stepTemplateObject.PSObject.Properties.Remove('LastModifiedBy')
					$stepTemplateObject.PSObject.Properties.Remove('LastModifiedOn')
					Write-Host $($stepTemplateObject | ConvertTo-Json -Depth 100)`),
				Update: strutil.StrPointer(strutil.StripMultilineWhitespace(`$host.ui.WriteErrorLine('Updating step template')
					$state = Read-Host | ConvertFrom-JSON
					if ([string]::IsNullOrEmpty($state.Id)) {
						$host.ui.WriteErrorLine('State ID is empty')
						Write-Host "{}"
					} else {
						$headers = @{ "X-Octopus-ApiKey" = $env:APIKEY }
						$body = Get-Content -Raw -Path ` + stepTemplateName + `.json

						# Replace feed IDs with lookup values passed in via env vars
						gci env:* | ? {$_.Name -like "FEED_*} | % {$body = $body.Replace($_.Name.Replace("FEED_", ""), $_.Value)}

						$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/$($env:SPACEID)/actiontemplates/$($state.Id)" -ContentType "application/json" -Method PUT -Body $body -Headers $headers
						# Strip out the last modified by details
						$stepTemplateObject = $response.content | ConvertFrom-Json
						$stepTemplateObject.PSObject.Properties.Remove('LastModifiedBy')
						$stepTemplateObject.PSObject.Properties.Remove('LastModifiedOn')
						Write-Host $($stepTemplateObject | ConvertTo-Json -Depth 100)
					}`)),
				Delete: strutil.StripMultilineWhitespace(`$host.ui.WriteErrorLine('Deleting step template')
					$state = Read-Host | ConvertFrom-JSON
					if ([string]::IsNullOrEmpty($state.Id)) {
						$host.ui.WriteErrorLine('State ID is empty')
					} else {
						$headers = @{ "X-Octopus-ApiKey" = $env:APIKEY }
						$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/$($env:SPACEID)/actiontemplates/$($state.Id)" -Method DELETE -Headers $headers
					}`),
			},
			Environment: environmentVars,
			SensitiveEnvironment: map[string]string{
				"SERVER":  "${var.octopus_server}",
				"SPACEID": "${var.octopus_space_id}",
				"APIKEY":  "${var.octopus_apikey}",
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
func (c StepTemplateConverter) writeData(file *hclwrite.File, resource octopus.StepTemplate, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c StepTemplateConverter) buildData(resourceName string, resource octopus.StepTemplate) terraform.TerraformExternalData {
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
			Write-Host ($keyValueResponse | ConvertTo-JSON -Depth 100)`},
		Query: map[string]string{
			"name":    resource.Name,
			"server":  "${var.octopus_server}",
			"apikey":  "${var.octopus_apikey}",
			"spaceid": "${var.octopus_space_id}",
		},
	}
}
