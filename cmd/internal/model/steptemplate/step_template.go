package steptemplate

type StepTemplate struct {
	Id                        string                   `json:"Id"`
	SpaceId                   *string                  `json:"SpaceId"`
	Version                   int                      `json:"Version"`
	Name                      string                   `json:"Name"`
	Description               string                   `json:"Description"`
	ActionType                string                   `json:"ActionType"`
	Packages                  []string                 `json:"Packages"`
	GitDependencies           []string                 `json:"GitDependencies"`
	Meta                      StepTemplateMeta         `json:"$Meta"`
	Properties                StepTemplateProperties   `json:"Properties"`
	LastModifiedBy            string                   `json:"LastModifiedBy"`
	Category                  string                   `json:"Category"`
	StepPackageId             string                   `json:"StepPackageId"`
	Parameters                []StepTemplateParameters `json:"Parameters"`
	CommunityActionTemplateId *string                  `json:"CommunityActionTemplateId"`
}

type StepTemplateMeta struct {
	ExportedAt     string `json:"ExportedAt"`
	OctopusVersion string `json:"OctopusVersion"`
	Type           string `json:"Type"`
}

type StepTemplateProperties struct {
	OctopusActionGoogleCloudUseVMServiceAccount        string `json:"Octopus.Action.GoogleCloud.UseVMServiceAccount"`
	OctopusActionGoogleCloudImpersonateServiceAccount  string `json:"Octopus.Action.GoogleCloud.ImpersonateServiceAccount"`
	OctopusActionTerraformGoogleCloudAccount           string `json:"Octopus.Action.Terraform.GoogleCloudAccount"`
	OctopusActionTerraformAzureAccount                 string `json:"Octopus.Action.Terraform.AzureAccount"`
	OctopusActionTerraformManagedAccount               string `json:"Octopus.Action.Terraform.ManagedAccount"`
	OctopusActionTerraformAllowPluginDownloads         string `json:"Octopus.Action.Terraform.AllowPluginDownloads"`
	OctopusActionScriptScriptSource                    string `json:"Octopus.Action.Script.ScriptSource"`
	OctopusActionTerraformRunAutomaticFileSubstitution string `json:"Octopus.Action.Terraform.RunAutomaticFileSubstitution"`
	OctopusActionTerraformPlanJsonOutput               string `json:"Octopus.Action.Terraform.PlanJsonOutput"`
	OctopusActionTerraformTemplate                     string `json:"Octopus.Action.Terraform.Template"`
	OctopusActionTerraformTemplateParameters           string `json:"Octopus.Action.Terraform.TemplateParameters"`
	OctopusActionRunOnServer                           string `json:"Octopus.Action.RunOnServer"`
	OctopusUseBundledTooling                           string `json:"OctopusUseBundledTooling"`
}

type StepTemplateParameters struct {
	Id              string                               `json:"Id"`
	Name            string                               `json:"Name"`
	Label           string                               `json:"Label"`
	HelpText        string                               `json:"HelpText"`
	DefaultValue    string                               `json:"DefaultValue"`
	DisplaySettings StepTemplateParameterDisplaySettings `json:"DisplaySettings"`
}

type StepTemplateParameterDisplaySettings struct {
	OctopusControlType string `json:"Octopus.ControlType"`
}
