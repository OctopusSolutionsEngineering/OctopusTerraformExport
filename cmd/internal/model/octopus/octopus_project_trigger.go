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

type ProjectTriggerSources struct {
	DeploymentActionSlug string
	GitDependencyName    string
	IncludeFilePaths     []string
	ExcludeFilePaths     []string
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
	Sources             []ProjectTriggerSources
	// WebhookId is the server generated identifier used in the URL that invokes a webhook trigger
	WebhookId *string
	// RequireApiKey indicates a webhook trigger is authenticated with an Octopus API key rather than a shared secret
	RequireApiKey *bool
	// Secret is the shared secret used to authenticate calls to a webhook trigger
	Secret *ProjectTriggerFilterSecret
}

// ProjectTriggerFilterSecret represents a webhook trigger's shared secret. The API only reports whether a
// secret has been set, and never returns the value itself.
type ProjectTriggerFilterSecret struct {
	HasValue bool
	NewValue *string
	Hint     *string
}

type ProjectTriggerFilterPackage struct {
	DeploymentActionSlug string
	DeploymentAction     string
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
