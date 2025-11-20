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

resource "octopusdeploy_variable" "string_variable" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "String"
  name      = "Test"
  value     = "PlainText"
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

resource "octopusdeploy_process" "test" {
  project_id = "${octopusdeploy_project.deploy_frontend_project.id}"
  depends_on = []
}

resource "octopusdeploy_process_steps_order" "test" {
  process_id = "${octopusdeploy_process.test.id}"
  steps      = ["${octopusdeploy_process_step.parent_step.id}"]
}

resource "octopusdeploy_process_step" "parent_step" {
  name                  = "Parent Step"
  type                  = "Placeholder"
  process_id            = "${octopusdeploy_process.test.id}"
  channels              = null
  condition             = "Variable"
  environments          = null
  excluded_environments = null
  package_requirement   = "LetOctopusDecide"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
    "Octopus.Step.ConditionVariableExpression" = "#{IsTrue}"
    "Octopus.Action.TargetRoles" = "cloud"
  }
}

resource "octopusdeploy_process_child_steps_order" "process_child_step_order_tfproviderv1_parent_step" {
  process_id = octopusdeploy_process.test.id
  parent_id  = octopusdeploy_process_step.parent_step.id
  children      = [
    octopusdeploy_process_child_step.process_child_step_test_child_step_1.id,
    octopusdeploy_process_child_step.process_child_step_test_child_step_2.id]
}

resource "octopusdeploy_process_child_step" "process_child_step_test_child_step_1" {
  name                  = "Child Step 1"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.test.id}"
  parent_id = octopusdeploy_process_step.parent_step.id
  channels              = null
  condition             = "Success"
  environments          = ["${octopusdeploy_environment.production_environment.id}"]
  excluded_environments = null
  is_required           = true
  notes                 = "notes"
  tenant_tags           = null
  execution_properties  = {
    "Octopus.Action.AutoRetry.MaximumCount" = "3"
    "Octopus.Action.AutoRetry.MinimumBackoff" = "15"
    "Octopus.Action.ExecutionTimeout.Minutes" = "10"
    "OctopusUseBundledTooling" = "False"
    "Octopus.Action.RunOnServer" = "true"
    "Octopus.Action.Script.ScriptSource" = "Inline"
    "Octopus.Action.Script.Syntax" = "PowerShell"
    "Octopus.Action.Script.ScriptBody" = "echo \"hi\""
  }
}

resource "octopusdeploy_process_child_step" "process_child_step_test_child_step_2" {
  name                  = "Child Step 2"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.test.id}"
  parent_id = octopusdeploy_process_step.parent_step.id
  channels              = null
  condition             = "Variable"
  environments          = ["${octopusdeploy_environment.development_environment.id}"]
  excluded_environments = null
  is_required           = true
  tenant_tags           = null
  execution_properties  = {
    "Octopus.Action.Script.ScriptBody" = "echo \"hi\""
    "Octopus.Action.AutoRetry.MaximumCount" = "3"
    "OctopusUseBundledTooling" = "False"
    "Octopus.Action.ExecutionTimeout.Minutes" = "10"
    "Octopus.Action.RunOnServer" = "true"
    "Octopus.Action.ConditionVariableExpression" = "#{IsTrue}"
    "Octopus.Action.Script.Syntax" = "PowerShell"
    "Octopus.Action.AutoRetry.MinimumBackoff" = "15"
    "Octopus.Action.Script.ScriptSource" = "Inline"
  }
}

resource "octopusdeploy_runbook" "runbook_runbookcac_test" {
  name                        = "Runbook"
  project_id                  = "${octopusdeploy_project.deploy_frontend_project.id}"
  environment_scope           = "All"
  environments                = []
  force_package_download      = false
  default_guided_failure_mode = "EnvironmentDefault"
  description                 = ""
  multi_tenancy_mode          = "Untenanted"

  retention_policy {
    quantity_to_keep = 100
  }

  connectivity_policy {
    allow_deployments_to_no_targets = true
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "None"
  }
}

resource "octopusdeploy_process" "process_runbookcac_test" {
  project_id = "${octopusdeploy_project.deploy_frontend_project.id}"
  runbook_id = "${octopusdeploy_runbook.runbook_runbookcac_test.id}"
  depends_on = []
}

resource "octopusdeploy_process_steps_order" "process_step_order_runbookcac_test" {
  process_id = "${octopusdeploy_process.process_runbookcac_test.id}"
  steps      = ["${octopusdeploy_process_step.process_step_runbookcac_test_parent_step.id}"]
}

resource "octopusdeploy_process_step" "process_step_runbookcac_test_parent_step" {
  name                  = "Parent Step"
  type                  = "Placeholder"
  process_id            = "${octopusdeploy_process.process_runbookcac_test.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  package_requirement   = "LetOctopusDecide"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
    "Octopus.Action.TargetRoles" = "eks"
  }
}

resource "octopusdeploy_process_child_steps_order" "process_child_step_order_runbookcac_test_parent_step" {
  process_id = "${octopusdeploy_process.process_runbookcac_test.id}"
  parent_id  = "${octopusdeploy_process_step.process_step_runbookcac_test_parent_step.id}"
  children      = ["${octopusdeploy_process_child_step.process_child_step_runbookcac_test_child_step_1.id}", "${octopusdeploy_process_child_step.process_child_step_runbookcac_test_child_step_2.id}"]
}

resource "octopusdeploy_process_child_step" "process_child_step_runbookcac_test_child_step_1" {
  name                  = "Child Step 1"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.process_runbookcac_test.id}"
  parent_id = octopusdeploy_process_step.process_step_runbookcac_test_parent_step.id
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  tenant_tags           = null
  execution_properties  = {
    "Octopus.Action.Script.ScriptBody" = "echo \"hi\""
    "Octopus.Action.Script.ScriptSource" = "Inline"
    "Octopus.Action.Script.Syntax" = "PowerShell"
    "OctopusUseBundledTooling" = "False"
    "Octopus.Action.RunOnServer" = "true"
  }
}

resource "octopusdeploy_process_child_step" "process_child_step_runbookcac_test_child_step_2" {
  name                  = "Child Step 2"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.process_runbookcac_test.id}"
  parent_id = octopusdeploy_process_step.process_step_runbookcac_test_parent_step.id
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  tenant_tags           = null
  execution_properties  = {
    "Octopus.Action.Script.ScriptBody" = "echo \"hi\""
    "Octopus.Action.Script.ScriptSource" = "Inline"
    "Octopus.Action.Script.Syntax" = "PowerShell"
    "OctopusUseBundledTooling" = "False"
    "Octopus.Action.RunOnServer" = "true"
  }
}