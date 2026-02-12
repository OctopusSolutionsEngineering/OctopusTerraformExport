package converters

import (
	"fmt"
	"strconv"

	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/boolutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/client"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/dummy"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/hcl"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/octopus"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/terraform"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/sanitizer"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/variables"
	"github.com/google/uuid"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const octopusdeployStepTemplateResourceType = "octopusdeploy_step_template"
const octopusdeployCommunityStepTemplateDataType = "octopusdeploy_community_step_template"
const octopusdeployCommunityStepTemplateResourceType = "octopusdeploy_community_step_template"
const octopusdeployStepTemplateDataType = "octopusdeploy_step_template"

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
	IncludeSpaceInPopulation   bool
	InlineVariableValues       bool
	DummySecretGenerator       dummy.DummySecretGenerator
	TerraformVariableWriter    variables.TerraformVariableWriter
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
	thisResource.Lookup = "${data." + octopusdeployStepTemplateDataType + "." + resourceName + ".step_template.id}"
	thisResource.VersionLookup = "${data." + octopusdeployStepTemplateDataType + "." + resourceName + ".step_template.version}"
	thisResource.VersionCurrent = strconv.Itoa(*template.Version)
	thisResource.ToHcl = func() (string, error) {
		terraformResource := c.buildData(resourceName, template)
		file := hclwrite.NewEmptyFile()
		block := gohcl.EncodeAsBlock(terraformResource, "data")
		hcl.WriteLifecyclePostCondition(block, "Failed to resolve an step template called \""+template.Name+"\". This resource must exist in the space before this Terraform configuration is applied.", "self.step_template != null")
		file.Body().AppendBlock(block)

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c StepTemplateConverter) AllToHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(false, dependencies) })
}

func (c StepTemplateConverter) AllToStatelessHcl(dependencies *data.ResourceDetailsCollection) {
	c.ErrGroup.Go(func() error { return c.allToHcl(true, dependencies) })
}

func (c StepTemplateConverter) ToHclById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, false, dependencies)
}

func (c StepTemplateConverter) ToHclStatelessById(id string, dependencies *data.ResourceDetailsCollection) error {
	return c.toHclById(id, true, dependencies)
}

func (c StepTemplateConverter) toHclById(id string, stateless bool, dependencies *data.ResourceDetailsCollection) error {
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

	return c.toHcl(resource, communityStepTemplate, stateless, dependencies)
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
	communityStepTemplateName := "communitysteptemplate_" + sanitizer.SanitizeName(template.Name)

	/*if c.GenerateImportScripts {
		c.toBashImport(stepTemplateName, target.Name, dependencies)
		c.toPowershellImport(stepTemplateName, target.Name, dependencies)
	}*/

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
		if thisResource.ExternalID == "" {
			thisResource.VersionLookup = "${data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ".step_template != null " +
				"? data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ".step_template.version " +
				": " + octopusdeployStepTemplateResourceType + "." + stepTemplateName + "[0].version}"
			thisResource.Lookup = "${data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ".step_template != null " +
				"? data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ".step_template.id " +
				": " + octopusdeployStepTemplateResourceType + "." + stepTemplateName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployStepTemplateResourceType + "." + stepTemplateName + "}"
		} else {
			/*
				In stateless mode, we either find the existing step template installed by the community step template,
				or we reference the community step template resource.

				This is a little different to most resources where a data source for the resource is used to determine
				if it exists or not. Community step templates are different in that we always assume they exist
				(because they are synced from an external library), so the octopusdeploy_community_step_template
				data source will always return the details of any valid community step. But, community step templates are
				not always installed in the space.

				It is the presence of a step template with the same name as the community step template that indicates
				it has been installed.
			*/
			thisResource.VersionLookup = "${data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ".step_template != null " +
				"? data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ".step_template.version " +
				": " + octopusdeployCommunityStepTemplateResourceType + "." + communityStepTemplateName + "[0].version}"
			thisResource.Lookup = "${data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ".step_template != null " +
				"? data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ".step_template.id " +
				": " + octopusdeployCommunityStepTemplateResourceType + "." + communityStepTemplateName + "[0].id}"
			thisResource.Dependency = "${" + octopusdeployCommunityStepTemplateResourceType + "." + communityStepTemplateName + "}"
		}
	} else {
		if thisResource.ExternalID == "" {
			thisResource.Lookup = "${" + octopusdeployStepTemplateResourceType + "." + stepTemplateName + ".id}"
			thisResource.VersionLookup = "${" + octopusdeployStepTemplateResourceType + "." + stepTemplateName + ".version}"
		} else {
			thisResource.Lookup = "${" + octopusdeployCommunityStepTemplateResourceType + "." + communityStepTemplateName + ".id}"
			thisResource.VersionLookup = "${" + octopusdeployCommunityStepTemplateResourceType + "." + communityStepTemplateName + ".version}"
		}
	}

	thisResource.ToHcl = func() (string, error) {

		file := hclwrite.NewEmptyFile()

		if thisResource.ExternalID != "" {
			// We need to query the community step template by its external URL to get its ID
			communityStepTemplateData := terraform.TerraformCommunityStepTemplateData{
				Name:    communityStepTemplateName,
				Type:    octopusdeployCommunityStepTemplateDataType,
				Website: strutil.NilIfEmpty(thisResource.ExternalID),
			}

			communityStepTemplateDataBlock := gohcl.EncodeAsBlock(communityStepTemplateData, "data")

			file.Body().AppendBlock(communityStepTemplateDataBlock)

			// We then need to reference the ID of the community step template data source in the step template resource
			communityStepTemplateResource := terraform.TerraformCommunityStepTemplate{
				Type: octopusdeployCommunityStepTemplateResourceType,
				Name: communityStepTemplateName,
				// It is possible the community step template is not installed or the id/website is not valid, so we need to check if the steps array has any entries
				CommunityActionTemplateId: "${length(data." + octopusdeployCommunityStepTemplateDataType + "." + communityStepTemplateName + ".steps) != 0 ? " +
					"data." + octopusdeployCommunityStepTemplateDataType + "." + communityStepTemplateName + ".steps[0].id : null}",
			}

			if stateless {
				c.writeData(file, template, stepTemplateName)
				/*
					When the step template is stateless, the resource is created if the data source does not return any results.
					We measure the presence of results by the length of the keys of the result attribute of the data source.
				*/
				communityStepTemplateResource.Count = strutil.StrPointer("${length(data." + octopusdeployCommunityStepTemplateDataType + "." + communityStepTemplateName + ".steps) != 0 ? 0 : 1}")
			}

			communityStepTemplateResourceBlock := gohcl.EncodeAsBlock(communityStepTemplateResource, "resource")

			file.Body().AppendBlock(communityStepTemplateResourceBlock)
		} else {

			terraformResource := terraform.TerraformStepTemplate{
				Type:                      octopusdeployStepTemplateResourceType,
				Name:                      stepTemplateName,
				ActionType:                template.ActionType,
				SpaceId:                   strutil.InputIfEnabled(c.IncludeSpaceInPopulation, dependencies.GetResourceDependency("Spaces", strutil.EmptyIfNil(template.SpaceId))),
				ResourceName:              template.Name,
				Description:               strutil.TrimPointer(template.Description), // The API trims whitespace, which can lead to a "Provider produced inconsistent result after apply" error
				StepPackageId:             template.StepPackageId,
				CommunityActionTemplateId: nil,
				Packages:                  c.convertPackages(template.Packages),
				Parameters:                c.convertParameters(template.Parameters, file, dependencies),
				Properties:                c.convertStepProperties(template.Properties),
			}

			if stateless {
				c.writeData(file, template, stepTemplateName)
				/*
					When the step template is stateless, the resource is created if the data source does not return any results.
					We measure the presence of results by the length of the keys of the result attribute of the data source.
				*/
				terraformResource.Count = strutil.StrPointer("${data." + octopusdeployStepTemplateDataType + "." + stepTemplateName + ".step_template != null ? 0 : 1}")
			}

			block := gohcl.EncodeAsBlock(terraformResource, "resource")

			if stateless {
				hcl.WriteLifecyclePreventDestroyAttribute(block)
			}

			file.Body().AppendBlock(block)
		}

		return string(file.Bytes()), nil
	}

	dependencies.AddResource(thisResource)

	return nil
}

func (c StepTemplateConverter) convertStepProperties(properties map[string]string) map[string]string {
	// "Octopus.Action.RunOnServer" might be set on the step template, but it is not returned
	// again when the step template is created. So we remove it here.
	return lo.OmitByKeys(properties, []string{"Octopus.Action.RunOnServer"})
}

func (c StepTemplateConverter) convertParameters(parameters []octopus.StepTemplateParameters, file *hclwrite.File, dependencies *data.ResourceDetailsCollection) []terraform.TerraformStepTemplateParameter {
	return lo.Map(parameters, func(item octopus.StepTemplateParameters, index int) terraform.TerraformStepTemplateParameter {
		/*
			The TF provider requires a UUID for the ID. However, it is possible that the ID is null or an empty string
			on the Octopus server. If we get a blank string, just generate an ID.
		*/

		id := item.Id
		if id == "" {
			id = uuid.New().String()
		}

		template := terraform.TerraformStepTemplateParameter{
			Id:       id,
			Name:     item.Name,
			Label:    strutil.NilIfEmpty(item.Label),
			HelpText: strutil.NilIfEmpty(item.HelpText),
			DisplaySettings: map[string]string{
				"Octopus.ControlType": item.DisplaySettings.OctopusControlType,
			},
		}

		if strutil.IsString(item.DefaultValue) {
			template.DefaultValue = strutil.NilIfEmpty(fmt.Sprint(item.DefaultValue))
		} else {
			var sensitiveValue *string = nil
			if !c.InlineVariableValues {
				sensitiveValue = c.TerraformVariableWriter.WriteTerraformVariablesForSecret(c.GetResourceType(), file, &item, dependencies)
			} else {
				sensitiveValue = strutil.StrPointer("\"" + *c.DummySecretGenerator.GetDummySecret() + "\"")
			}
			template.DefaultSensitiveValue = sensitiveValue
		}

		return template
	})
}

func (c StepTemplateConverter) convertPackages(packages []octopus.Package) []terraform.TerraformStepTemplatePackage {
	return lo.Map(packages, func(item octopus.Package, index int) terraform.TerraformStepTemplatePackage {
		return terraform.TerraformStepTemplatePackage{
			Name:                    strutil.EmptyIfNil(item.Name),
			PackageID:               item.PackageId,
			AcquisitionLocation:     item.AcquisitionLocation,
			ExtractDuringDeployment: boolutil.NilIfFalse(item.ExtractDuringDeployment),
			FeedId:                  strutil.EmptyIfNil(item.FeedId),
			Properties:              c.convertProperties(item.Properties),
		}
	})
}

func (c StepTemplateConverter) convertProperties(properties map[string]string) terraform.TerraformStepTemplatePackageProperties {
	extract, ok := properties["Extract"]
	if !ok {
		extract = ""
	}

	selectionMode, ok := properties["SelectionMode"]
	if !ok {
		selectionMode = ""
	}

	packageParameterName, ok := properties["PackageParameterName"]
	if !ok {
		packageParameterName = ""
	}

	purpose, ok := properties["Purpose"]
	if !ok {
		purpose = ""
	}

	return terraform.TerraformStepTemplatePackageProperties{
		SelectionMode:        selectionMode,
		Extract:              extract,
		PackageParameterName: packageParameterName,
		Purpose:              purpose,
	}
}

// writeData appends the data blocks for stateless modules
func (c StepTemplateConverter) writeData(file *hclwrite.File, resource octopus.StepTemplate, resourceName string) {
	terraformResource := c.buildData(resourceName, resource)
	block := gohcl.EncodeAsBlock(terraformResource, "data")
	file.Body().AppendBlock(block)
}

func (c StepTemplateConverter) buildData(resourceName string, resource octopus.StepTemplate) terraform.TerraformStepTemplateData {

	return terraform.TerraformStepTemplateData{
		Type:         octopusdeployStepTemplateDataType,
		Name:         resourceName,
		SpaceId:      nil,
		Id:           nil,
		ResourceName: strutil.NilIfEmpty(resource.Name),
	}
}
