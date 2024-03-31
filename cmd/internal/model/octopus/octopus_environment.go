package octopus

type Environment struct {
	NameId
	SpaceId                    string
	Description                *string
	SortOrder                  int
	UseGuidedFailure           bool
	AllowDynamicInfrastructure bool
	ExtensionSettings          []Extension
}

type Extension struct {
	ExtensionId string
	Values      map[string]interface{}
}
