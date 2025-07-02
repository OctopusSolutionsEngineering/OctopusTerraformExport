variable "runbook_every_step_project_example_runbook_name" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The name of the runbook exported from Example Runbook"
  default     = "Example Runbook"
}
resource "octopusdeploy_runbook" "runbook_every_step_project_example_runbook" {
  name                        = "${var.runbook_every_step_project_example_runbook_name}"
  project_id                  = "${octopusdeploy_project.project_every_step_project.id}"
  environment_scope           = "Specified"
  environments                = ["${octopusdeploy_environment.environment_development.id}"]
  force_package_download      = false
  default_guided_failure_mode = "EnvironmentDefault"
  description                 = "This is an example of a runbook"
  multi_tenancy_mode          = "TenantedOrUntenanted"

  retention_policy {
    quantity_to_keep = 100
  }

  connectivity_policy {
    allow_deployments_to_no_targets = true
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "None"
  }
}
