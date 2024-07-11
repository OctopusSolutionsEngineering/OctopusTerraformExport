package terraform

import "github.com/OctopusSolutionsEngineering/OctopusTerraformExport/cmd/internal/strutil"

type TerraformConfig struct {
	RequiredProviders RequiredProviders `hcl:"required_providers,block"`
	Backend           *Backend          `hcl:"backend,block"`
	RequiredVersion   *string           `hcl:"required_version"`
}

type Backend struct {
	Type string `hcl:"type,label"`
}

type RequiredProviders struct {
	OctopusProvider OctopusProvider `hcl:"octopusdeploy"`
}

type OctopusProvider struct {
	Source  string `cty:"source"`
	Version string `cty:"version"`
}

type TerraformProvider struct {
	Type    string  `hcl:"type,label"`
	Address string  `hcl:"address"`
	ApiKey  string  `hcl:"api_key"`
	SpaceId *string `hcl:"space_id"`
}

func (c TerraformConfig) CreateTerraformConfig(backend string, version string) TerraformConfig {
	config := TerraformConfig{
		RequiredProviders: RequiredProviders{
			OctopusProvider: OctopusProvider{
				Source:  "OctopusDeployLabs/octopusdeploy",
				Version: strutil.DefaultIfEmpty(version, "0.21.5"),
			},
		},
		RequiredVersion: strutil.StrPointer(">= 1.6.0"),
	}

	if backend != "" {
		config.Backend = &Backend{Type: backend}
	}

	return config
}
