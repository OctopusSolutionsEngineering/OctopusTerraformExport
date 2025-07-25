package octopus

type CommunityStepTemplate struct {
	Id            string                   `json:"Id"`
	Name          string                   `json:"Name"`
	Type          string                   `json:"Type"`
	Author        string                   `json:"Author"`
	Description   string                   `json:"Description"`
	Website       string                   `json:"Website"`
	HistoryUrl    string                   `json:"HistoryUrl"`
	Properties    map[string]string        `json:"Properties"`
	Parameters    []StepTemplateParameters `json:"Parameters"`
	Version       int                      `json:"Version"`
	StepPackageId string                   `json:"StepPackageId"`
}

type StepTemplate struct {
	Id                        string                   `json:"Id"`
	SpaceId                   *string                  `json:"SpaceId"`
	Version                   *int                     `json:"Version"`
	Name                      string                   `json:"Name"`
	Description               *string                  `json:"Description"`
	ActionType                string                   `json:"ActionType"`
	Packages                  []Package                `json:"Packages"`
	GitDependencies           []string                 `json:"GitDependencies"`
	Properties                map[string]string        `json:"Properties"`
	LastModifiedBy            string                   `json:"LastModifiedBy"`
	Category                  string                   `json:"Category"`
	StepPackageId             string                   `json:"StepPackageId"`
	Parameters                []StepTemplateParameters `json:"Parameters"`
	CommunityActionTemplateId *string                  `json:"CommunityActionTemplateId"`
}

type StepTemplateParameters struct {
	Id              string                               `json:"Id"`
	Name            string                               `json:"Name"`
	Label           string                               `json:"Label"`
	HelpText        string                               `json:"HelpText"`
	DefaultValue    any                                  `json:"DefaultValue"`
	DisplaySettings StepTemplateParameterDisplaySettings `json:"DisplaySettings"`
}

type StepTemplateParameterDisplaySettings struct {
	OctopusControlType string `json:"Octopus.ControlType"`
}
