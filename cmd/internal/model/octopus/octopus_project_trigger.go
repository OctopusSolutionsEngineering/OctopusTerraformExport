package octopus

type ProjectTrigger struct {
	Id          string
	Name        string
	Description *string
	ProjectId   string
	IsDisabled  bool
	Filter      ProjectTriggerFilter
	Action      ProjectTriggerAction
}

type ProjectTriggerFilter struct {
	FilterType          string
	EnvironmentIds      []string
	Roles               []string
	EventGroups         []string
	EventCategories     []string
	DaysOfWeek          []string
	Timezone            *string
	Id                  *string
	LastModifiedOn      *string
	LastModifiedBy      *string
	Packages            []ProjectTriggerFilterPackage
	StartTime           *string
	MonthlyScheduleType *string
	DateOfMonth         *string
	DayNumberOfMonth    *string
	DayOfWeek           *string
	Interval            *string
	RunAfter            *string
	RunUntil            *string
	CronExpression      *string
	HourInterval        *int
	MinuteInterval      *int
}

type ProjectTriggerFilterPackage struct {
	DeploymentActionSlug string
	PackageReference     string
}

type ProjectTriggerAction struct {
	ActionType                                 string
	RunbookId                                  *string
	ShouldRedeployWhenMachineHasBeenDeployedTo bool
	Id                                         *string
	LastModifiedOn                             *string
	LastModifiedBy                             *string
	SourceEnvironmentIds                       []string
	EnvironmentIds                             []string
	DestinationEnvironmentId                   *string
	EnvironmentId                              *string
	ShouldRedeployWhenReleaseIsCurrent         *bool
	ChannelId                                  *string
	TenantIds                                  []string
	TenantTags                                 []string
}
