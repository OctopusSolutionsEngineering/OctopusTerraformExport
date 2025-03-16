package terraform

type TerraformDeploymentFreeze struct {
	Type              string                                      `hcl:"type,label"`
	Name              string                                      `hcl:"name,label"`
	Count             *string                                     `hcl:"count"`
	Id                *string                                     `hcl:"id"`
	ResourceName      string                                      `hcl:"name"`
	Start             string                                      `hcl:"start"`
	End               string                                      `hcl:"end"`
	RecurringSchedule *TerraformDeploymentFreezeRecurringSchedule `cty:"recurring_schedule"`
}

type TerraformDeploymentFreezeRecurringSchedule struct {
	EndType             string   `hcl:"end_type"`
	Type                string   `hcl:"type"`
	Unit                int      `hcl:"unit"`
	DateOfMonth         *string  `hcl:"date_of_month"`
	DayNumberOfMonth    *string  `hcl:"day_number_of_month"`
	DayOfWeek           *string  `hcl:"day_of_week"`
	DaysOfWeek          []string `hcl:"days_of_week"`
	EndAfterOccurrences *int     `hcl:"end_after_occurrences"`
	EndOnDate           *string  `hcl:"end_on_date"`
	MonthlyScheduleType *string  `hcl:"monthly_schedule_type"`
}
