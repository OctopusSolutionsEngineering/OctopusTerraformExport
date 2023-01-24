package terraform

type TerraformConfig struct {
	RequiredProviders RequiredProviders `hcl:"required_providers,block"`
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

func (c TerraformConfig) CreateTerraformConfig() TerraformConfig {
	return TerraformConfig{
		RequiredProviders: RequiredProviders{
			OctopusProvider: OctopusProvider{
				Source:  "OctopusDeployLabs/octopusdeploy",
				Version: "0.10.1",
			},
		},
	}
}
