package octopus

type DeploymentFreezes struct {
	DeploymentFreezes []DeploymentFreeze
	Count             int
}

type DeploymentFreeze struct {
	Id                            string
	Name                          string
	Start                         string
	End                           string
	TenantProjectEnvironmentScope []DeploymentFreezeTenantProjectEnvironmentScope
	ProjectEnvironmentScope       map[string][]string
	RecurringSchedule             *DeploymentFreezeRecurringSchedule
}

type DeploymentFreezeTenantProjectEnvironmentScope struct {
	TenantId      string
	ProjectId     string
	EnvironmentId string
}

type DeploymentFreezeRecurringSchedule struct {
	Type                string
	Unit                int
	EndType             string
	EndOnDate           *string
	EndAfterOccurrences *int
	StartDate           *string
	EndDate             *string
	DateOfMonth         *string
	DayNumberOfMonth    *string
	DayOfWeek           *string
	DaysOfWeek          []string
	MonthlyScheduleType *string
}
