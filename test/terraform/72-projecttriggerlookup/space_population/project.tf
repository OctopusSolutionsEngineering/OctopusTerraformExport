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

data "octopusdeploy_project_groups" "project_group" {
  ids          = null
  partial_name = "Test"
  skip         = 0
  take         = 1
}

data "octopusdeploy_feeds" "docker_feed" {
  feed_type    = "Docker"
  ids          = null
  partial_name = ""
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
      feed_id = "${data.octopusdeploy_feeds.docker_feed.feeds[0].id}",
      id = null,
      package_id = "package1",
      properties = { Extract = "False", Purpose = "", SelectionMode = "immediate" }
    }
  }
  slug                  = "get-mysql-host"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
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
  environments                = [data.octopusdeploy_environments.dev.environments[0].id]
  default_guided_failure_mode = "EnvironmentDefault"
  force_package_download      = false
}

resource "octopusdeploy_process" "process_test_myrunbook3" {
  project_id = "${octopusdeploy_project.deploy_frontend_project.id}"
  runbook_id = "${octopusdeploy_runbook.runbook.id}"
  depends_on = []
}

resource "octopusdeploy_process_steps_order" "process_step_order_test_myrunbook3" {
  process_id = "${octopusdeploy_process.process_test_myrunbook3.id}"
  steps      = [
    "${octopusdeploy_process_step.process_step_test_myrunbook3_hello_world__using_powershell_.id}",
    "${octopusdeploy_process_step.process_step_test_myrunbook3_test.id}"]
}

resource "octopusdeploy_process_step" "process_step_test_myrunbook3_test" {
  name                  = "Test"
  type                  = "Octopus.TerraformApply"
  process_id            = "${octopusdeploy_process.process_test_myrunbook3.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  package_requirement   = "LetOctopusDecide"
  primary_package       = {
    acquisition_location = "Server",
    feed_id = "${data.octopusdeploy_feeds.built_in_feed.feeds[0].id}",
    id = null,
    package_id = "terraform",
    properties = { SelectionMode = "immediate"
    }
  }
  slug                  = "test"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
  }
  execution_properties  = {
    "Octopus.Action.Terraform.AllowPluginDownloads" = "True"
    "Octopus.Action.RunOnServer" = "True"
    "Octopus.Action.Terraform.GoogleCloudAccount" = "False"
    "Octopus.Action.GoogleCloud.ImpersonateServiceAccount" = "False"
    "Octopus.Action.Script.ScriptSource" = "Package"
    "Octopus.Action.GoogleCloud.UseVMServiceAccount" = "True"
    "Octopus.Action.Terraform.TemplateDirectory" = "blah"
    "Octopus.Action.Terraform.AzureAccount" = "False"
    "Octopus.Action.Terraform.ManagedAccount" = "None"
    "Octopus.Action.Terraform.PlanJsonOutput" = "False"
    "Octopus.Action.Terraform.RunAutomaticFileSubstitution" = "True"
  }
}

resource "octopusdeploy_process_step" "process_step_test_myrunbook3_hello_world__using_powershell_" {
  name                  = "Hello world (using PowerShell)"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.process_test_myrunbook3.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  is_required           = true
  package_requirement   = "LetOctopusDecide"
  packages              = {
    package1 = {
      acquisition_location = "Server",
      feed_id = "#{HelmFeed}",
      id = null,
      package_id = "package1",
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
    "Octopus.Action.Script.ScriptBody" = "Write-Host 'Hello world, using PowerShell'\n\n#TODO: Experiment with steps of your own :)\n\nWrite-Host '[Learn more about the types of steps available in Octopus](https://oc.to/OnboardingAddStepsLearnMore)'"
    "Octopus.Action.Script.ScriptSource" = "Inline"
    "Octopus.Action.Script.Syntax" = "PowerShell"
    "Octopus.Action.RunOnServer" = "True"
  }
}