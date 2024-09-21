package generators

import (
	"encoding/json"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/steptemplate"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"
	"github.com/google/uuid"
	"github.com/zeebo/xxh3"
	"strings"
	"time"
)

type StepTemplateGenerator struct {
}

func (s StepTemplateGenerator) Generate(collection *data.ResourceDetailsCollection, name string, stepKey string, description string) ([]byte, error) {
	templateText, err := s.createTemplate(collection)

	if err != nil {
		return nil, err
	}

	templateParams, err := s.createTerraformTemplateParameters(collection, stepKey)

	if err != nil {
		return nil, err
	}

	templateParamsJson, err := json.Marshal(templateParams)

	if err != nil {
		return nil, err
	}

	stepTemplateParams, err := s.createStepTemplateParameters(collection, stepKey)

	if err != nil {
		return nil, err
	}

	template := steptemplate.StepTemplate{
		Id:              s.createStableGuid(name),
		Name:            name,
		Description:     description,
		ActionType:      "Octopus.TerraformApply",
		Packages:        []string{},
		GitDependencies: []string{},
		Properties: steptemplate.StepTemplateProperties{
			OctopusActionGoogleCloudUseVMServiceAccount:        "False",
			OctopusActionGoogleCloudImpersonateServiceAccount:  "False",
			OctopusActionTerraformGoogleCloudAccount:           "False",
			OctopusActionTerraformAzureAccount:                 "False",
			OctopusActionTerraformManagedAccount:               "None",
			OctopusActionTerraformAllowPluginDownloads:         "True",
			OctopusActionScriptScriptSource:                    "Inline",
			OctopusActionTerraformRunAutomaticFileSubstitution: "True",
			OctopusActionTerraformPlanJsonOutput:               "False",
			OctopusActionTerraformTemplate:                     strutil.UnEscapeDollar(templateText),
			OctopusActionTerraformTemplateParameters:           string(templateParamsJson[:]),
			OctopusActionRunOnServer:                           "True",
			OctopusUseBundledTooling:                           "False",
		},
		LastModifiedBy: "OctopusDeploy",
		Category:       "octopus",
		StepPackageId:  "Octopus.TerraformApply",
		Parameters:     stepTemplateParams,
		Version:        1,
		Meta: steptemplate.StepTemplateMeta{
			ExportedAt:     time.Now().Format(time.RFC3339),
			OctopusVersion: "2024.1.10177",
			Type:           "ActionTemplate",
		},
	}

	return json.MarshalIndent(template, "", "\t")
}

func (s StepTemplateGenerator) createStableGuid(name string) string {
	h := xxh3.HashString128(name).Bytes()
	guid, _ := uuid.FromBytes(h[:])
	return guid.String()
}

func (s StepTemplateGenerator) createTemplate(collection *data.ResourceDetailsCollection) (string, error) {
	sb := strings.Builder{}
	for _, resource := range collection.Resources {
		// Some resources are already resolved by their parent, but exist in the resource details map as a lookup.
		// In these cases, ToHcl is nil.
		if resource.ToHcl == nil {
			continue
		}

		// Do not export supporting files like bash or powershell scripts
		if !strings.HasSuffix(resource.FileName, ".tf") {
			continue
		}

		hcl, err := resource.ToHcl()

		if err != nil {
			return "", err
		}

		sb.WriteString(hcl + "\n")
	}

	return sb.String(), nil
}

func (s StepTemplateGenerator) createTerraformTemplateParameters(collection *data.ResourceDetailsCollection, stepKey string) (map[string]string, error) {

	parameters := map[string]string{}

	// These are the common parameters exposed by all octoterra modules
	parameters["octopus_server"] = "#{ReferenceArchitecture." + stepKey + ".Octopus.ServerUrl}"
	parameters["octopus_apikey"] = "#{ReferenceArchitecture." + stepKey + ".Octopus.ApiKey}"
	parameters["octopus_space_id"] = "#{ReferenceArchitecture." + stepKey + ".Octopus.SpaceId}"

	for _, resource := range collection.Resources {
		for _, parameter := range resource.Parameters {
			name := "ReferenceArchitecture." + stepKey + "." + resource.ResourceType + "." + parameter.ResourceName + "." + parameter.ParameterType
			parameters[parameter.VariableName] = "#{" + name + "}"
		}
	}

	return parameters, nil
}

func (s StepTemplateGenerator) createStepTemplateParameters(collection *data.ResourceDetailsCollection, stepKey string) ([]steptemplate.StepTemplateParameters, error) {

	parameters := []steptemplate.StepTemplateParameters{}

	// These are the common parameters exposed by all octoterra modules
	parameters = append(parameters, steptemplate.StepTemplateParameters{
		Id:           s.createStableGuid("ReferenceArchitecture." + stepKey + ".Octopus.ServerUrl"),
		Name:         "ReferenceArchitecture." + stepKey + ".Octopus.ServerUrl",
		Label:        "Octopus Server URL",
		HelpText:     "The Octopus server URL.",
		DefaultValue: "#{Octopus.Web.ServerUri}",
		DisplaySettings: steptemplate.StepTemplateParameterDisplaySettings{
			OctopusControlType: "SingleLineText",
		},
	})
	parameters = append(parameters, steptemplate.StepTemplateParameters{
		Id:           s.createStableGuid("ReferenceArchitecture." + stepKey + ".Octopus.ApiKey"),
		Name:         "ReferenceArchitecture." + stepKey + ".Octopus.ApiKey",
		Label:        "Octopus API Key",
		HelpText:     "The Octopus API key. See the [Octopus docs](https://octopus.com/docs/octopus-rest-api/how-to-create-an-api-key) for more details on creating an API Key.",
		DefaultValue: "",
		DisplaySettings: steptemplate.StepTemplateParameterDisplaySettings{
			OctopusControlType: "Sensitive",
		},
	})
	parameters = append(parameters, steptemplate.StepTemplateParameters{
		Id:           s.createStableGuid("ReferenceArchitecture." + stepKey + ".Octopus.SpaceId"),
		Name:         "ReferenceArchitecture." + stepKey + ".Octopus.SpaceId",
		Label:        "Octopus Space ID",
		HelpText:     "The Octopus space ID.",
		DefaultValue: "#{Octopus.Space.Id}",
		DisplaySettings: steptemplate.StepTemplateParameterDisplaySettings{
			OctopusControlType: "SingleLineText",
		},
	})

	for _, resource := range collection.Resources {
		for _, parameter := range resource.Parameters {
			name := "ReferenceArchitecture." + stepKey + "." + resource.ResourceType + "." + parameter.ResourceName + "." + parameter.ParameterType
			parameters = append(parameters, steptemplate.StepTemplateParameters{
				Id:           s.createStableGuid(name),
				Name:         name,
				Label:        parameter.Label,
				HelpText:     parameter.Description,
				DefaultValue: parameter.DefaultValue,
				DisplaySettings: steptemplate.StepTemplateParameterDisplaySettings{
					OctopusControlType: s.getControlType(parameter),
				},
			})
		}
	}

	return parameters, nil
}

func (s StepTemplateGenerator) getControlType(parameter data.ResourceParameter) string {
	if parameter.Sensitive {
		return "Sensitive"
	}

	return "SingleLineText"
}
