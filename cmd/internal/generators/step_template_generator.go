package generators

import (
	"encoding/json"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/data"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/model/steptemplate"
	"github.com/google/uuid"
	"github.com/zeebo/xxh3"
	"strings"
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
			OctopusActionTerraformTemplate:                     templateText,
			OctopusActionTerraformTemplateParameters:           string(templateParamsJson[:]),
			OctopusActionRunOnServer:                           "True",
			OctopusUseBundledTooling:                           "False",
		},
		LastModifiedBy: "OctopusDeploy",
		Category:       "octopus",
		Parameters:     stepTemplateParams,
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

		hcl, err := resource.ToHcl()

		if err != nil {
			return "", err
		}

		sb.WriteString(hcl + "\\n")
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
			parameters[parameter.Name] = "#{ReferenceArchitecture." + stepKey + "." + parameter.Type + "}"
		}
	}

	return parameters, nil
}

func (s StepTemplateGenerator) createStepTemplateParameters(collection *data.ResourceDetailsCollection, stepKey string) ([]steptemplate.StepTemplateParameters, error) {

	parameters := []steptemplate.StepTemplateParameters{}

	// These are the common parameters exposed by all octoterra modules
	parameters = append(parameters, steptemplate.StepTemplateParameters{
		Id:           s.createStableGuid("ReferenceArchitecture." + stepKey + ".Octopus.ServerUrl"),
		Name:         "ReferenceArchitecture.Octopus.ServerUrl",
		Label:        "Octopus Server URL",
		HelpText:     "The Octopus server URL.",
		DefaultValue: "",
		DisplaySettings: steptemplate.StepTemplateParameterDisplaySettings{
			OctopusControlType: "SingleLineText",
		},
	})
	parameters = append(parameters, steptemplate.StepTemplateParameters{
		Id:           s.createStableGuid("ReferenceArchitecture." + stepKey + ".Octopus.ApiKey"),
		Name:         "ReferenceArchitecture.Octopus.ApiKey",
		Label:        "Octopus API Key",
		HelpText:     "The Octopus API key. See the [Octopus docs](https://octopus.com/docs/octopus-rest-api/how-to-create-an-api-key) for more details on creating an API Key.",
		DefaultValue: "",
		DisplaySettings: steptemplate.StepTemplateParameterDisplaySettings{
			OctopusControlType: "Sensitive",
		},
	})
	parameters = append(parameters, steptemplate.StepTemplateParameters{
		Id:           s.createStableGuid("ReferenceArchitecture." + stepKey + ".Octopus.SpaceId"),
		Name:         "ReferenceArchitecture.Octopus.SpaceId",
		Label:        "Octopus Space ID",
		HelpText:     "The Octopus space ID.",
		DefaultValue: "",
		DisplaySettings: steptemplate.StepTemplateParameterDisplaySettings{
			OctopusControlType: "SingleLineText",
		},
	})

	for _, resource := range collection.Resources {
		for _, parameter := range resource.Parameters {
			parameters = append(parameters, steptemplate.StepTemplateParameters{
				Id:           s.createStableGuid("#{ReferenceArchitecture." + stepKey + "." + parameter.Type + "}"),
				Name:         "#{ReferenceArchitecture." + parameter.Type + "}",
				Label:        parameter.Label,
				HelpText:     parameter.Description,
				DefaultValue: "",
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
