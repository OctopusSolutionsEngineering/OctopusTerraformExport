package terraform

type TerraformProjectScheduledTrigger struct {
	Type                      string                                                     `hcl:"type,label"`
	Name                      string                                                     `hcl:"name,label"`
	Count                     *string                                                    `hcl:"count"`
	Id                        *string                                                    `hcl:"id"`
	SpaceId                   *string                                                    `hcl:"space_id"`
	ResourceName              string                                                     `hcl:"name"`
	Description               *string                                                    `hcl:"description"`
	ProjectId                 string                                                     `hcl:"project_id"`
	DeployNewReleaseAction    *TerraformProjectScheduledTriggerDeployNewReleaseAction    `hcl:"deploy_new_release_action,block"`
	OnceDailySchedule         *TerraformProjectScheduledTriggerDaily                     `hcl:"once_daily_schedule,block"`
	ContinuousDailySchedule   *TerraformProjectScheduledTriggerContinuousDailySchedule   `hcl:"continuous_daily_schedule,block"`
	CronExpressionSchedule    *TerraformProjectScheduledTriggerCronExpressionSchedule    `hcl:"cron_expression_schedule,block"`
	RunRunbookAction          *TerraformProjectScheduledTriggerRunRunbookAction          `hcl:"run_runbook_action,block"`
	DeployLatestReleaseAction *TerraformProjectScheduledTriggerDeployLatestReleaseAction `hcl:"deploy_latest_release_action,block"`
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
	MinuteInterval int      `hcl:"minute_interval "`
	DaysOfWeek     []string `hcl:"days_of_week"`
}
