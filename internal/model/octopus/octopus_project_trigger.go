package octopus

type ProjectTrigger struct {
	Id         string
	Name       string
	SpaceId    *string
	ProjectId  string
	IsDisabled bool
	Filter     ProjectTriggerFilter
	Action     ProjectTriggerAction
}

type ProjectTriggerFilter struct {
	FilterType      string
	EnvironmentIds  []string
	Roles           []string
	EventGroups     []string
	EventCategories []string
	Id              *string
	LastModifiedOn  *string
	LastModifiedBy  *string
}

type ProjectTriggerAction struct {
	ActionType                                 string
	ShouldRedeployWhenMachineHasBeenDeployedTo bool
	Id                                         *string
	LastModifiedOn                             *string
	LastModifiedBy                             *string
}
