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
	OctopusProvider  ProviderDefinition  `hcl:"octopusdeploy"`
	ShellProvider    *ProviderDefinition `hcl:"shell"`
	ExternalProvider *ProviderDefinition `hcl:"external"`
}

type ProviderDefinition struct {
	Source  string `cty:"source"`
	Version string `cty:"version"`
}

type TerraformProvider struct {
	Type    string  `hcl:"type,label"`
	Address *string `hcl:"address"`
	ApiKey  *string `hcl:"api_key"`
	SpaceId *string `hcl:"space_id"`
}

type TerraformShellProvider struct {
	Type              string   `hcl:"type,label"`
	Interpreter       []string `hcl:"interpreter"`
	EnableParallelism bool     `hcl:"enable_parallelism"`
}

type TerraformEmptyProvider struct {
	Type string `hcl:"type,label"`
}

func (c TerraformConfig) CreateTerraformConfig(backend string, version string, experimentalStepTemplateEnabled bool) TerraformConfig {
	config := TerraformConfig{
		RequiredProviders: RequiredProviders{
			OctopusProvider: ProviderDefinition{
				Source:  "OctopusDeploy/octopusdeploy",
				Version: strutil.DefaultIfEmpty(version, "1.0.1"),
			},
		},
		RequiredVersion: strutil.StrPointer(">= 1.6.0"),
	}

	if experimentalStepTemplateEnabled {
		config.RequiredProviders.ShellProvider = &ProviderDefinition{
			Source:  "scottwinkler/shell",
			Version: "1.7.10",
		}
		config.RequiredProviders.ExternalProvider = &ProviderDefinition{
			Source:  "hashicorp/external",
			Version: "2.3.4",
		}
	}

	if backend != "" {
		config.Backend = &Backend{Type: backend}
	}

	return config
}
