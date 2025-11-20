data "octopusdeploy_feeds" "built_in_feed" {
  feed_type    = "BuiltIn"
  ids          = null
  partial_name = ""
  skip         = 0
  take         = 1
}

data "octopusdeploy_lifecycles" "lifecycle_default_lifecycle" {
  ids          = null
  partial_name = "Default Lifecycle"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_variable" "workerpool_variable" {
  owner_id = octopusdeploy_project.deploy_frontend_project.id
  type     = "String"
  name     = "WorkerPool"
  value    = octopusdeploy_static_worker_pool.static_pool.id
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
  steps      = ["${octopusdeploy_process_step.process_step_get_mysql_host.id}"]
}

resource "octopusdeploy_process_step" "process_step_get_mysql_host" {
  name                  = "Get MySQL Host"
  type                  = "Octopus.KubernetesRunScript"
  process_id            = "${octopusdeploy_process.test.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  package_requirement   = "LetOctopusDecide"
  packages              = {
    package1 = {
      acquisition_location = "ExecutionTarget",
      feed_id = "${octopusdeploy_docker_container_registry.feed_docker.id}",
      id = null,
      package_id = "package1",
      properties = {
        Extract = "False",
        Purpose = "",
        SelectionMode = "immediate"
      }
    }
  }
  slug                  = "get-mysql-host"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  worker_pool_variable  = "#{WorkerPool}"
  execution_properties  = {
    "Octopus.Action.Script.Syntax" = "PowerShell"
    "Octopus.Action.Script.ScriptBody" = "echo \"hi\""
    "Octopus.Action.RunOnServer" = "true"
    "Octopus.Action.Script.ScriptSource" = "Inline"
  }
  properties            = {
    "Octopus.Action.TargetRoles" = "eks"
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
  environments                = []
  default_guided_failure_mode = "EnvironmentDefault"
  force_package_download      = false
}

resource "octopusdeploy_process" "process_test_runbook" {
  project_id = octopusdeploy_project.deploy_frontend_project.id
  runbook_id = "${octopusdeploy_runbook.runbook.id}"
  depends_on = []
}

resource "octopusdeploy_process_steps_order" "process_step_order_test_runbook" {
  process_id = "${octopusdeploy_process.process_test_runbook.id}"
  steps      = ["${octopusdeploy_process_step.process_step_test_runbook_hello_world__using_powershell_.id}"]
}

variable "project_runbook_step_hello_world__using_powershell__package_package1_packageid" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The package ID for the package named package1 from step Hello world (using PowerShell) in project Runbook"
  default     = "package1"
}

resource "octopusdeploy_process_step" "process_step_test_runbook_hello_world__using_powershell_" {
  name                  = "Hello world (using PowerShell)"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.process_test_runbook.id}"
  channels              = null
  condition             = "Success"
  environments          = []
  excluded_environments = null
  is_required           = true
  package_requirement   = "LetOctopusDecide"
  worker_pool_variable  = "#{WorkerPool}"
  packages              = {
    package1 = {
      acquisition_location = "Server",
      feed_id = "${data.octopusdeploy_feeds.built_in_feed.feeds[0].id}",
      id = null,
      package_id = "${var.project_runbook_step_hello_world__using_powershell__package_package1_packageid}",
      properties = {
        Extract = "True",
        Purpose = "",
        SelectionMode = "immediate"
      }
    }
  }
  slug                  = "hello-world-using-powershell"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
  }
  execution_properties  = {
    "Octopus.Action.RunOnServer" = "true"
    "Octopus.Action.Script.ScriptBody" = "Write-Host 'Hello world, using PowerShell'\n\n#TODO: Experiment with steps of your own :)\n\nWrite-Host '[Learn more about the types of steps available in Octopus](https://oc.to/OnboardingAddStepsLearnMore)'"
    "Octopus.Action.Script.ScriptSource" = "Inline"
    "Octopus.Action.Script.Syntax" = "PowerShell"
    "Octopus.Action.EnabledFeatures" = "Octopus.Features.JsonConfigurationVariables"
  }
}