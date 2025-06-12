data "octopusdeploy_lifecycles" "lifecycle_default_lifecycle" {
  ids          = null
  partial_name = "Default Lifecycle"
  skip         = 0
  take         = 1
}

data "octopusdeploy_worker_pools" "workerpool_default" {
  name = "Default Worker Pool"
  ids  = null
  skip = 0
  take = 1
}

data "octopusdeploy_feeds" "built_in_feed" {
  feed_type    = "BuiltIn"
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

resource "octopusdeploy_process" "process_tfapply" {
  project_id = "${octopusdeploy_project.deploy_frontend_project.id}"
  depends_on = []
}

resource "octopusdeploy_process_steps_order" "process_step_order_tfapply" {
  process_id = "${octopusdeploy_process.process_tfapply.id}"
  steps      = ["${octopusdeploy_process_step.process_step_tfapply_test.id}"]
}

resource "octopusdeploy_process_step" "process_step_tfapply_test" {
  name                  = "Test"
  type                  = "Octopus.TerraformApply"
  process_id            = "${octopusdeploy_process.process_tfapply.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  package_requirement   = "LetOctopusDecide"
  slug                  = "test"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  properties            = {
  }
  execution_properties  = {
    "Octopus.Action.Terraform.RunAutomaticFileSubstitution" = "True"
    "Octopus.Action.GoogleCloud.ImpersonateServiceAccount" = "False"
    "Octopus.Action.Terraform.ManagedAccount" = "None"
    "Octopus.Action.Terraform.GoogleCloudAccount" = "False"
    "Octopus.Action.RunOnServer" = "true"
    "Octopus.Action.Script.ScriptSource" = "Inline"
    "Octopus.Action.Terraform.TemplateParameters" = jsonencode({        })
    "Octopus.Action.Terraform.AllowPluginDownloads" = "True"
    "Octopus.Action.Terraform.Template" = "#test"
    "Octopus.Action.Terraform.AzureAccount" = "False"
    "Octopus.Action.GoogleCloud.UseVMServiceAccount" = "True"
    "Octopus.Action.Terraform.PlanJsonOutput" = "False"
  }
}