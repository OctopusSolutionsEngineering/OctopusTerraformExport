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

func (s StepTemplateGenerator) Generate(collection *data.ResourceDetailsCollection, name string, description string) ([]byte, error) {
	templateText, err := s.createTemplate(collection)

	if err != nil {
		return nil, err
	}

	templateParams, err := s.createTemplateParameters(collection)

	if err != nil {
		return nil, err
	}

	templateParamsJson, err := json.Marshal(templateParams)

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
	}

	return json.Marshal(template)
}

func (s StepTemplateGenerator) createStableGuid(name string) string {
	h := xxh3.HashString128(name).Bytes()
	guid, _ := uuid.FromBytes(h[:])
	return guid.String()
}

func (s StepTemplateGenerator) createTemplate(collection *data.ResourceDetailsCollection) (string, error) {
	var sb strings.Builder
	for _, resource := range collection.Resources {
		hcl, err := resource.ToHcl()

		if err != nil {
			return "", err
		}

		sb.WriteString(hcl + "\\n")
	}

	return sb.String(), nil
}

func (s StepTemplateGenerator) createTemplateParameters(collection *data.ResourceDetailsCollection) (map[string]string, error) {
	return map[string]string{}, nil
}
