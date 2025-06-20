resource "octopusdeploy_project_scheduled_trigger" "projecttrigger_every_step_project_scheduled_trigger" {
  space_id    = "${trimspace(var.octopus_space_id)}"
  name        = "Scheduled Trigger"
  description = "This is an example of a runbook scheduled trigger"
  timezone    = "UTC"
  is_disabled = false
  project_id  = "${octopusdeploy_project.project_every_step_project.id}"
  tenant_ids  = ["${octopusdeploy_tenant.tenant_australian_office.id}"]

  once_daily_schedule {
    start_time   = "2025-05-07T09:00:00"
    days_of_week = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"]
  }

  run_runbook_action {
    target_environment_ids = ["${octopusdeploy_environment.environment_development.id}"]
    runbook_id             = "${octopusdeploy_runbook.runbook_every_step_project_example_runbook.id}"
  }
}
