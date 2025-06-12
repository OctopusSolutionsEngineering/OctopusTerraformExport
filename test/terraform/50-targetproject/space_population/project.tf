resource "octopusdeploy_project" "project_1" {
  auto_create_release                  = false
  default_guided_failure_mode          = "EnvironmentDefault"
  default_to_skip_if_already_installed = false
  description                          = "Test project"
  discrete_channel_release             = false
  is_disabled                          = false
  is_discrete_channel_release          = false
  is_version_controlled                = false
  lifecycle_id                         = octopusdeploy_lifecycle.simple_lifecycle.id
  name                                 = "Test"
  project_group_id                     = octopusdeploy_project_group.project_group_test.id
  tenanted_deployment_participation    = "Untenanted"
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

resource "octopusdeploy_variable" "scoped_var" {
  owner_id = octopusdeploy_project.project_1.id
  type     = "String"
  name     = "test"
  value    = "test"
  scope {
    machines = [
      octopusdeploy_azure_service_fabric_cluster_deployment_target.target_service_fabric.id,
      octopusdeploy_azure_web_app_deployment_target.target_web_app.id,
      octopusdeploy_cloud_region_deployment_target.target_region1.id,
      octopusdeploy_kubernetes_cluster_deployment_target.test_eks.id,
      octopusdeploy_listening_tentacle_deployment_target.listening.id,
      octopusdeploy_offline_package_drop_deployment_target.target_offlineoffline.id,
      octopusdeploy_polling_tentacle_deployment_target.polling.id,
      octopusdeploy_ssh_connection_deployment_target.ssh.id
    ]
  }

  depends_on = [octopusdeploy_process.test]
}

output "octopus_project_1" {
  value = octopusdeploy_project.project_1.id
}

resource "octopusdeploy_project" "project_2" {
  auto_create_release                  = false
  default_guided_failure_mode          = "EnvironmentDefault"
  default_to_skip_if_already_installed = false
  description                          = "Test project 2"
  discrete_channel_release             = false
  is_disabled                          = false
  is_discrete_channel_release          = false
  is_version_controlled                = false
  lifecycle_id                         = octopusdeploy_lifecycle.simple_lifecycle2.id
  name                                 = "Test 2"
  project_group_id                     = octopusdeploy_project_group.project_group_test.id
  tenanted_deployment_participation    = "Untenanted"
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
  project_id = "${octopusdeploy_project.project_1.id}"
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