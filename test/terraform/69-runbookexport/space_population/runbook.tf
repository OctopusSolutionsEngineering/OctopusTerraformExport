data "octopusdeploy_feeds" "built_in_feed" {
  feed_type    = "BuiltIn"
  ids          = null
  partial_name = ""
  skip         = 0
  take         = 1
}

data "octopusdeploy_projects" "example" {
  partial_name           = "Test"
  skip                   = 0
  take                   = 1
}

resource "octopusdeploy_runbook" "runbook" {
  project_id         = data.octopusdeploy_projects.example.projects[0].id
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
  project_id = "${data.octopusdeploy_projects.example.projects[0].id}"
  runbook_id = "${octopusdeploy_runbook.runbook.id}"
  depends_on = []
}

resource "octopusdeploy_process_steps_order" "process_step_order_test_runbook" {
  process_id = "${octopusdeploy_process.process_test_runbook.id}"
  steps      = ["${octopusdeploy_process_step.process_step_test_runbook_hello_world__using_powershell_.id}", "${octopusdeploy_process_step.process_step_test_runbook_test.id}"]
}

variable "project_runbook_step_test_packageid" {
  type        = string
  nullable    = false
  sensitive   = false
  description = "The package ID for the package named  from step Test in project Runbook"
  default     = "terraform"
}
resource "octopusdeploy_process_step" "process_step_test_runbook_test" {
  name                  = "Test"
  type                  = "Octopus.TerraformApply"
  process_id            = "${octopusdeploy_process.process_test_runbook.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  package_requirement   = "LetOctopusDecide"
  primary_package       = {
    acquisition_location = "Server",
    feed_id = "${data.octopusdeploy_feeds.built_in_feed.feeds[0].id}",
    id = null,
    package_id = "${var.project_runbook_step_test_packageid}",
    properties = {
      SelectionMode = "immediate"
    }
  }
  slug                  = "test"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
  }
  execution_properties  = {
    "Octopus.Action.GoogleCloud.ImpersonateServiceAccount" = "False"
    "Octopus.Action.Script.ScriptSource" = "Package"
    "Octopus.Action.Terraform.GoogleCloudAccount" = "False"
    "Octopus.Action.RunOnServer" = "True"
    "Octopus.Action.Terraform.TemplateDirectory" = "blah"
    "Octopus.Action.Terraform.AllowPluginDownloads" = "True"
    "Octopus.Action.Terraform.AzureAccount" = "False"
    "Octopus.Action.GoogleCloud.UseVMServiceAccount" = "True"
    "Octopus.Action.Terraform.ManagedAccount" = "None"
    "Octopus.Action.Terraform.RunAutomaticFileSubstitution" = "True"
    "Octopus.Action.Terraform.PlanJsonOutput" = "False"
  }
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
    "Octopus.Action.RunOnServer" = "True"
    "Octopus.Action.Script.ScriptBody" = "Write-Host 'Hello world, using PowerShell'\n\n#TODO: Experiment with steps of your own :)\n\nWrite-Host '[Learn more about the types of steps available in Octopus](https://oc.to/OnboardingAddStepsLearnMore)'"
    "Octopus.Action.Script.ScriptSource" = "Inline"
    "Octopus.Action.Script.Syntax" = "PowerShell"
    "Octopus.Action.EnabledFeatures" = "Octopus.Features.JsonConfigurationVariables"
  }
}