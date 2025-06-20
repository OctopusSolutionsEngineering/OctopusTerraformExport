variable "project_every_step_project_name" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The name of the project exported from Every Step Project"
  default     = "Every Step Project"
}
variable "project_every_step_project_description_prefix" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "An optional prefix to add to the project description for the project Every Step Project"
  default     = ""
}
variable "project_every_step_project_description_suffix" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "An optional suffix to add to the project description for the project Every Step Project"
  default     = ""
}
variable "project_every_step_project_description" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The description of the project exported from Every Step Project"
  default     = "This sample project has every step in Octopus assigned to the deployment process. These steps can be used as examples on which to build custom projects."
}
variable "project_every_step_project_tenanted" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The tenanted setting for the project TenantedOrUntenanted"
  default     = "TenantedOrUntenanted"
}
resource "octopusdeploy_project" "project_every_step_project" {
  name                                 = "${var.project_every_step_project_name}"
  auto_create_release                  = false
  default_guided_failure_mode          = "EnvironmentDefault"
  default_to_skip_if_already_installed = false
  discrete_channel_release             = false
  is_disabled                          = false
  is_version_controlled                = false
  lifecycle_id                         = "${octopusdeploy_lifecycle.lifecycle_application.id}"
  project_group_id                     = "${data.octopusdeploy_project_groups.project_group_default_project_group.project_groups[0].id}"
  included_library_variable_sets       = ["${octopusdeploy_library_variable_set.library_variable_set_variables_example_variable_set.id}"]
  tenanted_deployment_participation    = "${var.project_every_step_project_tenanted}"

  template {
    name             = "Example.Tenant.Variable"
    label            = "An example tenant variable required to be defined by all tenants that deploy this project."
    help_text        = "This is where the help text associated with the variable is defined."
    default_value    = "The default value"
    display_settings = { "Octopus.ControlType" = "MultiLineText" }
  }

  connectivity_policy {
    allow_deployments_to_no_targets = true
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "None"
  }

  versioning_strategy {
    template = "#{Octopus.Version.LastMajor}.#{Octopus.Version.LastMinor}.#{Octopus.Version.NextPatch}"
  }

  lifecycle {
    ignore_changes = ["git_username_password_persistence_settings[0].password"]
  }
  description = "${var.project_every_step_project_description_prefix}${var.project_every_step_project_description}${var.project_every_step_project_description_suffix}"
}
