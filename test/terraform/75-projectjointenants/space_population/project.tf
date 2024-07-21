data "octopusdeploy_lifecycles" "lifecycle_default_lifecycle" {
  ids          = null
  partial_name = "Default Lifecycle"
  skip         = 0
  take         = 1
}

data "octopusdeploy_feeds" "built_in_feed" {
  feed_type    = "BuiltIn"
  ids          = null
  partial_name = ""
  skip         = 0
  take         = 1
}

data "octopusdeploy_project_groups" "project_group" {
  ids          = null
  partial_name = "Test"
  skip         = 0
  take         = 1
}

data "octopusdeploy_library_variable_sets" "variable" {
  partial_name = "Octopus Variables"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_project" "deploy_frontend_project" {
  auto_create_release                  = false
  default_guided_failure_mode          = "EnvironmentDefault"
  default_to_skip_if_already_installed = false
  description                          = "Test project"
  discrete_channel_release             = false
  is_disabled                          = false
  is_discrete_channel_release          = false
  is_version_controlled                = false
  lifecycle_id                         = data.octopusdeploy_lifecycles.lifecycle_default_lifecycle.lifecycles[0].id
  name                                 = "Test"
  project_group_id                     = data.octopusdeploy_project_groups.project_group.project_groups[0].id
  tenanted_deployment_participation    = "TenantedOrUntenanted"
  space_id                             = var.octopus_space_id
  included_library_variable_sets       = [data.octopusdeploy_library_variable_sets.variable.library_variable_sets[0].id]
  versioning_strategy {
    template = "#{Octopus.Version.LastMajor}.#{Octopus.Version.LastMinor}.#{Octopus.Version.LastPatch}.#{Octopus.Version.NextRevision}"
  }

  connectivity_policy {
    allow_deployments_to_no_targets = false
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "SkipUnavailableMachines"
  }
  template {
    name             = "Project Template Variable"
    label            = "Test"
    default_value    = "Test"
    display_settings = { "Octopus.ControlType" = "SingleLineText" }
  }
}

output "lvs_common_template_id" {
  value = tolist([for tmp in data.octopusdeploy_library_variable_sets.variable.library_variable_sets[0].template : tmp.id if tmp.name == "template"])[0]
}

output "lvs_common_template_id_2" {
  value = "${data.octopusdeploy_library_variable_sets.variable.library_variable_sets[0].template_ids[\"template\"]}"
}