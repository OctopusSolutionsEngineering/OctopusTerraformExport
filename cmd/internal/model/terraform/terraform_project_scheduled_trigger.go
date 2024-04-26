package terraform

type TerraformProjectScheduledTrigger struct {
	Type                      string                                                     `hcl:"type,label"`
	Name                      string                                                     `hcl:"name,label"`
	Count                     *string                                                    `hcl:"count"`
	Id                        *string                                                    `hcl:"id"`
	SpaceId                   *string                                                    `hcl:"space_id"`
	ResourceName              string                                                     `hcl:"name"`
	Description               *string                                                    `hcl:"description"`
	Timezone                  *string                                                    `hcl:"timezone"`
	IsDisabled                bool                                                       `hcl:"is_disabled"`
	ChannelId                 *string                                                    `hcl:"channel_id"`
	ProjectId                 string                                                     `hcl:"project_id"`
	TenantIds                 []string                                                   `hcl:"tenant_ids"`
	DeployNewReleaseAction    *TerraformProjectScheduledTriggerDeployNewReleaseAction    `hcl:"deploy_new_release_action,block"`
	OnceDailySchedule         *TerraformProjectScheduledTriggerDaily                     `hcl:"once_daily_schedule,block"`
	ContinuousDailySchedule   *TerraformProjectScheduledTriggerContinuousDailySchedule   `hcl:"continuous_daily_schedule,block"`
	CronExpressionSchedule    *TerraformProjectScheduledTriggerCronExpressionSchedule    `hcl:"cron_expression_schedule,block"`
	RunRunbookAction          *TerraformProjectScheduledTriggerRunRunbookAction          `hcl:"run_runbook_action,block"`
	DeployLatestReleaseAction *TerraformProjectScheduledTriggerDeployLatestReleaseAction `hcl:"deploy_latest_release_action,block"`
	DaysPerMonthSchedule      *TerraformProjectScheduledTriggerDaysPerMonthSchedule      `hcl:"days_per_month_schedule,block"`
}

type TerraformProjectScheduledTriggerDeployNewReleaseAction struct {
	DestinationEnvironmentId string `hcl:"destination_environment_id"`
}

type TerraformProjectScheduledTriggerDeployLatestReleaseAction struct {
	SourceEnvironmentId      string `hcl:"source_environment_id"`
	DestinationEnvironmentId string `hcl:"destination_environment_id"`
	ShouldRedeploy           bool   `hcl:"should_redeploy"`
}

type TerraformProjectScheduledTriggerDaily struct {
	StartTime  string   `hcl:"start_time"`
	DaysOfWeek []string `hcl:"days_of_week"`
}

type TerraformProjectScheduledTriggerDaysPerMonthSchedule struct {
	MonthlyScheduleType string `hcl:"monthly_schedule_type"`
	StartTime           string `hcl:"start_time"`
	DateOfMonth         string `hcl:"date_of_month"`
	DayNumberOfMonth    string `hcl:"day_number_of_month"`
	DayOfWeek           string `hcl:"day_of_week"`
}

type TerraformProjectScheduledTriggerRunRunbookAction struct {
	TargetEnvironmentIds []string `hcl:"target_environment_ids"`
	RunbookId            string   `hcl:"runbook_id"`
}

type TerraformProjectScheduledTriggerCronExpressionSchedule struct {
	CronExpression string `hcl:"cron_expression"`
}

type TerraformProjectScheduledTriggerContinuousDailySchedule struct {
	Interval       string   `hcl:"interval"`
	RunAfter       string   `hcl:"run_after"`
	RunUntil       string   `hcl:"run_until"`
	HourInterval   int      `hcl:"hour_interval"`
	MinuteInterval int      `hcl:"minute_interval"`
	DaysOfWeek     []string `hcl:"days_of_week"`
}
