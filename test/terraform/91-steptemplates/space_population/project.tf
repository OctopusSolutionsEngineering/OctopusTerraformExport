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

resource "octopusdeploy_step_template" "steptemplate_hello_world" {
  action_type     = "Octopus.Script"
  name            = "Hello World"
  step_package_id = "Octopus.Script"
  packages        = []
  parameters      = [
    {
      default_value = "World!",
      display_settings = { "Octopus.ControlType" = "SingleLineText" },
      help_text = null,
      id = "fb95b2e8-3395-4b63-9c23-549c133841ab",
      label = null,
      name = "HelloWorld.Message"
    },
    {
      default_value = "SecretValue",
      display_settings = { "Octopus.ControlType" = "Sensitive" },
      help_text = null,
      id = "ca5b66cc-c859-407b-b4df-d6bab42ad2f1",
      label = null,
      name = "HelloWorld.Secret"
    }
  ]
  properties      = { "Octopus.Action.Script.ScriptBody" = "echo \"Hello #{HelloWorld.Message}\"", "Octopus.Action.Script.ScriptSource" = "Inline", "Octopus.Action.Script.Syntax" = "PowerShell" }
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

resource "octopusdeploy_process" "process_step_template" {
  project_id = "${octopusdeploy_project.deploy_frontend_project.id}"
  depends_on = []
}

resource "octopusdeploy_process_steps_order" "process_step_order_step_template" {
  process_id = "${octopusdeploy_process.process_step_template.id}"
  steps      = ["${octopusdeploy_process_templated_step.process_step_step_template_hello_world.id}"]
}

resource "octopusdeploy_process_templated_step" "process_step_step_template_hello_world" {
  name                  = "Hello World"
  process_id            = "${octopusdeploy_process.process_step_template.id}"
  template_id           = "${octopusdeploy_step_template.steptemplate_hello_world.id}"
  template_version      = "${octopusdeploy_step_template.steptemplate_hello_world.version}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  package_requirement   = "LetOctopusDecide"
  slug                  = "hello-world"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  worker_pool_id        = null
  parameters = {
    "HelloWorld.Message" = "there!"
  }
  properties            = {
  }
  execution_properties  = {
    "Octopus.Action.Script.Syntax" = "PowerShell"
    "Octopus.Action.Script.ScriptBody" = "echo \"Hello #{HelloWorld.Message}\""
    "Octopus.Action.RunOnServer" = "true"
    "Octopus.Action.Script.ScriptSource" = "Inline"
  }
}