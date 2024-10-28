package converters

import (
	"encoding/base64"
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

func (c StepTemplateConverter) ToHclLookupById(id string, dependencies *data.ResourceDetailsCollection) error {
	if id == "" {
		return nil
	}

	if dependencies.HasResource(id, c.GetResourceType()) {
		return nil
	}

	template := octopus.StepTemplate{}
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &template)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.StepTemplate: %w", err)
	}

	if c.Excluder.IsResourceExcludedWithRegex(template.Name, c.ExcludeAllStepTemplates, c.ExcludeStepTemplates, c.ExcludeStepTemplatesRegex, c.ExcludeStepTemplatesExcept) {
		return nil
	}

	// The first resource maps the step template name to the ID
	thisResource := data.ResourceDetails{}

	resourceName := "steptemplate_" + sanitizer.SanitizeName(template.Name)

	thisResource.FileName = "space_population/" + resourceName + ".tf"
	thisResource.Id = template.Id
	thisResource.Name = template.Name
	thisResource.ResourceType = c.GetResourceType()
	/*
		The result attribute of a data source is a map of key-value pairs. The key is the step template ID, and the value
		is the step template name. So the keys() is used to get the keys, and the only key is the step template ID.
	*/
	thisResource.Lookup = "${keys(data." + octopusdeployStepTemplateDataType + "." + resourceName + ".result)[0]}"
	thisResource.VersionLookup = "${values(data." + octopusdeployStepTemplateDataType + "." + resourceName + "_versions)[0]}"
	thisResource.VersionCurrent = strconv.Itoa(*template.Version)
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, template)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve an step template called \""+template.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(keys(self.result)) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	// The second resource maps the step template name to the version
	thisVersionsResource := data.ResourceDetails{}
	thisVersionsResource.FileName = "space_population/" + resourceName + "_versions.tf"
	thisVersionsResource.Id = template.Id
	thisVersionsResource.Name = template.Name
	thisVersionsResource.ResourceType = c.GetResourceType() + "_Versions"
	thisVersionsResource.Lookup = "${keys(data." + octopusdeployStepTemplateDataType + "." + resourceName + ".result)[0]}"
	thisVersionsResource.VersionLookup = "${values(data." + octopusdeployStepTemplateDataType + "." + resourceName + "_versions.result)[0]}"
	thisVersionsResource.VersionCurrent = strconv.Itoa(*template.Version)
	thisVersionsResource.ToHcl = func() (string, error) {
		terraformResource := c.buildDataVersions(resourceName+"_versions", template)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve an step template called \""+template.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "length(keys(self.result)) != 0")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisVersionsResource)
	return nil
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

	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
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
	_, err := c.Client.GetSpaceResourceById(c.GetResourceType(), id, &resource)

	if err != nil {
		return fmt.Errorf("error in OctopusClient.GetSpaceResourceById loading type octopus.StepTemplate: %w", err)
	}

	zap.L().Info("Step Template: " + resource.Id + " " + resource.Name)

	var communityStepTemplate *octopus.CommunityStepTemplate = nil
	if resource.CommunityActionTemplateId != nil {
		communityStepTemplate = &octopus.CommunityStepTemplate{}
		_, err := c.Client.GetGlobalResourceById("CommunityActionTemplates", strutil.EmptyIfNil(resource.CommunityActionTemplateId), communityStepTemplate)
		if err != nil {
			return err
		}
	}

	return c.toHcl(resource, communityStepTemplate, false, dependencies)
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

		zap.L().Info("Step Template: " + resource.Id + " " + resource.Name)

		var communityStepTemplate *octopus.CommunityStepTemplate = nil
		if resource.CommunityActionTemplateId != nil {
			communityStepTemplate = &octopus.CommunityStepTemplate{}
			_, err := c.Client.GetGlobalResourceById("CommunityActionTemplates", strutil.EmptyIfNil(resource.CommunityActionTemplateId), communityStepTemplate)
			if err != nil {
				return err
			}
		}

		err := c.toHcl(resource, communityStepTemplate, stateless, dependencies)

		if err != nil {
			return err
		}
	}

	return nil
}

func (c StepTemplateConverter) toHcl(template octopus.StepTemplate, communityStepTemplate *octopus.CommunityStepTemplate, stateless bool, dependencies *data.ResourceDetailsCollection) error {
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

	// Get the external ID, defined as the community step template website
	externalId := ""
	if communityStepTemplate != nil {
		externalId = communityStepTemplate.Website
	}

	thisResource := data.ResourceDetails{}
	thisResource.FileName = "space_population/" + stepTemplateName + ".tf"
	thisResource.Id = template.Id
	thisResource.Name = template.Name
	thisResource.VersionCurrent = strconv.Itoa(*template.Version)
	thisResource.ExternalID = externalId
	thisResource.ResourceType = c.GetResourceType()

	if stateless {
		thisResource.VersionLookup = "${length(keys(data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ")) != 0 " +
			"? values(data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + "_versions)[0] " +
			": " + octopusdeployStepTemplateResourceType + "." + stepTemplateName + "[0].output.Version}"
		thisResource.Lookup = "${length(keys(data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ")) != 0 " +
			"? values(data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ")[0] " +
			": " + octopusdeployStepTemplateResourceType + "." + stepTemplateName + "[0].output.Id}"
		thisResource.Dependency = "${" + octopusdeployStepTemplateResourceType + "." + stepTemplateName + "}"
	} else {
		thisResource.Lookup = "${" + octopusdeployStepTemplateResourceType + "." + stepTemplateName + ".output.Id}"
		thisResource.VersionLookup = "${" + octopusdeployStepTemplateResourceType + "." + stepTemplateName + ".output.Version}"
	}

	thisResource.ToHcl = func() (string, error) {

		// Step templates can reference feeds, which are space specific. We pass in lookup values for the feeds in
		// environment variables to allow the PowerShell scripts to replace and hard coded feed IDs.
		// Changes to environment variables trigger the script resource to run an update, so we also track the version
		// of the upstream step template here as well. The version is not used in the script, but forces updates
		// when the step template is updated.
		environmentVars := map[string]string{}
		environmentVars["VERSION"] = thisResource.VersionCurrent
		for _, v2 := range dependencies.GetAllResource("Feeds") {
			environmentVars["FEED_"+v2.Id] = v2.Lookup
		}

		stepTemplateBody := "Get-Content -Raw -Path " + stepTemplateName + ".json"

		if stateless {
			stepTemplateJson, err := json.Marshal(template)
			if err != nil {
				return "", err
			}

			stepTemplateJsonEncoded := base64.StdEncoding.EncodeToString(stepTemplateJson)
			stepTemplateBody = "[System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String(\"" + stepTemplateJsonEncoded + "\"))"
		}

		terraformResource := terraform.TerraformShellScript{
			Type: octopusdeployStepTemplateResourceType,
			Name: stepTemplateName,
			LifecycleCommands: terraform.TerraformShellScriptLifecycleCommands{
				Read: strutil.StrPointer(strutil.StripMultilineWhitespace(`$host.ui.WriteErrorLine('Reading step template')
					$state = [Console]::In.ReadLine() | ConvertFrom-JSON
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
					$body = ` + stepTemplateBody + `
					$parsedTemplate = $body | ConvertFrom-Json -Depth 100
	
					$response = $null
					if (-not [string]::IsNullOrEmpty($parsedTemplate.CommunityActionTemplateId)) {
						# Find the step template with the matching external ID
						$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/communityactiontemplates?take=10000" -Method GET -Headers $headers
						$communityTemplate = $response.content | ConvertFrom-Json | Select-Object -Expand Items | ? {$_.Website -eq "` + thisResource.ExternalID + `"} | % {
							# Then install the step template
							$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/communityactiontemplates/$($_.Id)/installation/$($env:SPACEID)" -Method POST -Headers $headers
						}
					} else {
						# Regular step templates are imported from their JSON representation

						# Replace feed IDs with lookup values passed in via env vars
						gci env:* | ? {$_.Name -like "FEED_*"} | % {$body = $body.Replace($_.Name.Replace("FEED_", ""), $_.Value)}
	
						$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/$($env:SPACEID)/actiontemplates" -ContentType "application/json" -Method POST -Body $body -Headers $headers
						$stepTemplate = $response.content | ConvertFrom-Json
						# Import any new step template twice to ensure the version of a new template is at least 1.
						$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/$($env:SPACEID)/actiontemplates/$($stepTemplate.Id)" -ContentType "application/json" -Method PUT -Body $body -Headers $headers
					}

					# Strip out the last modified by details
					$stepTemplateObject = $response.content | ConvertFrom-Json
					$stepTemplateObject.PSObject.Properties.Remove('LastModifiedBy')
					$stepTemplateObject.PSObject.Properties.Remove('LastModifiedOn')
					Write-Host $($stepTemplateObject | ConvertTo-Json -Depth 100)`),
				Update: strutil.StrPointer(strutil.StripMultilineWhitespace(`$host.ui.WriteErrorLine('Updating step template')
					$state = [Console]::In.ReadLine() | ConvertFrom-JSON
					if ([string]::IsNullOrEmpty($state.Id)) {
						$host.ui.WriteErrorLine('State ID is empty')
						Write-Host "{}"
					} else {
						$headers = @{ "X-Octopus-ApiKey" = $env:APIKEY }
						$body = ` + stepTemplateBody + `

						# Replace feed IDs with lookup values passed in via env vars
						gci env:* | ? {$_.Name -like "FEED_*"} | % {$body = $body.Replace($_.Name.Replace("FEED_", ""), $_.Value)}

						$response = Invoke-WebRequest -Uri "$($env:SERVER)/api/$($env:SPACEID)/actiontemplates/$($state.Id)" -ContentType "application/json" -Method PUT -Body $body -Headers $headers
						# Strip out the last modified by details
						$stepTemplateObject = $response.content | ConvertFrom-Json
						$stepTemplateObject.PSObject.Properties.Remove('LastModifiedBy')
						$stepTemplateObject.PSObject.Properties.Remove('LastModifiedOn')
						Write-Host $($stepTemplateObject | ConvertTo-Json -Depth 100)
					}`)),
				Delete: strutil.StripMultilineWhitespace(`$host.ui.WriteErrorLine('Deleting step template')
					$state = [Console]::In.ReadLine() | ConvertFrom-JSON
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
			/*
				When the step template is stateless, the resource is created if the data source does not return any results.
				We measure the presence of results by the length of the keys of the result attribute of the data source.
			*/
			terraformResource.Count = strutil.StrPointer("${length(keys(data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ".result)) != 0 ? 0 : 1}")
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

// writeData appends the data blocks for stateless modules
func (c StepTemplateConverter) writeData(file *hclwrite.File, resource octopus.StepTemplate, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)

	terraformResourceVersions := c.buildDataVersions(resourceName+"_versions", resource)
	blockVersions := gohcl.EncodeAsBlock(terraformResourceVersions, "data")
	file.Body().AppendBlock(blockVersions)
}

func (c StepTemplateConverter) buildData(resourceName string, resource octopus.StepTemplate) terraform.TerraformExternalData {
	/*
		Use Powershell to query the action templates.

		I've noticed this happening occasionally when running the script. I don't think it's a problem with the script,
		but may be specific to pwsh on Linux. There doesn't appear to be any solution other that retrying the terraform
		apply operation:

		The data source received an unexpected error while attempting to execute the
		program.

		The program was executed, however it returned no additional error messaging.

		Program: /opt/microsoft/powershell/7/pwsh
		State: signal: segmentation fault (core dumped)
	*/

	return terraform.TerraformExternalData{
		Type: octopusdeployStepTemplateDataType,
		Name: resourceName,
		Program: []string{
			"pwsh",
			"-Command",
			strutil.StripMultilineWhitespace(`
				$query = [Console]::In.ReadLine() | ConvertFrom-JSON
				$headers = @{ "X-Octopus-ApiKey" = $query.apikey }
				$response = Invoke-WebRequest -Uri "$($query.server)/api/$($query.spaceid)/actiontemplates?take=10000" -Method GET -Headers $headers
				$keyValueResponse = @{}
				$response.content | ConvertFrom-JSON | Select-Object -Expand Items | ? {$_.Name -eq $query.name} | % {$keyValueResponse[$_.Id] = $_.Name} | Out-Null
				$results = $keyValueResponse | ConvertTo-JSON -Depth 100
				Write-Host $results`)},
		Query: map[string]string{
			"name":    resource.Name,
			"server":  "${var.octopus_server}",
			"apikey":  "${var.octopus_apikey}",
			"spaceid": "${var.octopus_space_id}",
		},
	}
}

func (c StepTemplateConverter) buildDataVersions(resourceName string, resource octopus.StepTemplate) terraform.TerraformExternalData {
	/*
		Use Powershell to query the action templates.

		I've noticed this happening occasionally when running the script. I don't think it's a problem with the script,
		but may be specific to pwsh on Linux. There doesn't appear to be any solution other that retrying the terraform
		apply operation:

		The data source received an unexpected error while attempting to execute the
		program.

		The program was executed, however it returned no additional error messaging.

		Program: /opt/microsoft/powershell/7/pwsh
		State: signal: segmentation fault (core dumped)
	*/

	return terraform.TerraformExternalData{
		Type: octopusdeployStepTemplateDataType,
		Name: resourceName,
		Program: []string{
			"pwsh",
			"-Command",
			strutil.StripMultilineWhitespace(`
				$query = [Console]::In.ReadLine() | ConvertFrom-JSON
				$headers = @{ "X-Octopus-ApiKey" = $query.apikey }
				$response = Invoke-WebRequest -Uri "$($query.server)/api/$($query.spaceid)/actiontemplates?take=10000" -Method GET -Headers $headers
				$keyValueResponse = @{}
				$response.content | ConvertFrom-JSON | Select-Object -Expand Items | ? {$_.Name -eq $query.name} | % {$keyValueResponse[$_.Id] = $_.Version.ToString()} | Out-Null
				$results = $keyValueResponse | ConvertTo-JSON -Depth 100
				Write-Host $results`)},
		Query: map[string]string{
			"name":    resource.Name,
			"server":  "${var.octopus_server}",
			"apikey":  "${var.octopus_apikey}",
			"spaceid": "${var.octopus_space_id}",
		},
	}
}
