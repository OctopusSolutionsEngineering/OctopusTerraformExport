data "octopusdeploy_lifecycles" "lifecycle_default_lifecycle" {
  ids          = null
  partial_name = "Default Lifecycle"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_project" "deploy_frontend_project" {
  auto_create_release                  = false
  default_guided_failure_mode          = "EnvironmentDefault"
  default_to_skip_if_already_installed = false
  description                          = "Test project"

  is_disabled                          = false
  is_discrete_channel_release          = false
  is_version_controlled                = false
  lifecycle_id                         = data.octopusdeploy_lifecycles.lifecycle_default_lifecycle.lifecycles[0].id
  name                                 = "Test"
  project_group_id                     = octopusdeploy_project_group.project_group_test.id
  tenanted_deployment_participation    = "Untenanted"
  space_id                             = var.octopus_space_id
  included_library_variable_sets       = []
  versioning_strategy {
    template = "#{Octopus.Version.LastMajor}.#{Octopus.Version.LastMinor}.#{Octopus.Version.LastPatch}.#{Octopus.Version.NextRevision}"
  }

  connectivity_policy {
    allow_deployments_to_no_targets = false
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "SkipUnavailableMachines"
  }
}

resource "octopusdeploy_runbook" "runbook" {
  project_id         = octopusdeploy_project.deploy_frontend_project.id
  name               = "Runbook"
  description        = "Test Runbook"
  multi_tenancy_mode = "Untenanted"
  connectivity_policy {
    allow_deployments_to_no_targets = false
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "SkipUnavailableMachines"
  }
  retention_policy {
    quantity_to_keep = 10
  }
  environment_scope           = "Specified"
  environments                = [octopusdeploy_environment.development_environment.id]
  default_guided_failure_mode = "EnvironmentDefault"
  force_package_download      = false
}

resource "octopusdeploy_process" "process_test_runbook" {
  project_id = "${octopusdeploy_project.deploy_frontend_project.id}"
  runbook_id = "${octopusdeploy_runbook.runbook.id}"
  depends_on = []
}

resource "octopusdeploy_process_steps_order" "process_step_order_test_runbook" {
  process_id = "${octopusdeploy_process.process_test_runbook.id}"
  steps      = ["${octopusdeploy_process_step.process_step_test_runbook_hello_world__using_powershell.id}"]
}

resource "octopusdeploy_process_step" "process_step_test_runbook_hello_world__using_powershell" {
  name                  = "Hello world (using PowerShell)"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.process_test_runbook.id}"
  channels              = null
  condition             = "Success"
  excluded_environments = null
  is_required           = true
  package_requirement   = "LetOctopusDecide"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
  }
  execution_properties  = {
    "Octopus.Action.RunOnServer" = "True"
    "Octopus.Action.Script.ScriptBody" = "Write-Host \"Hello world\""
    "Octopus.Action.Script.ScriptSource" = "Inline"
    "Octopus.Action.Script.Syntax" = "PowerShell"
  }
}
