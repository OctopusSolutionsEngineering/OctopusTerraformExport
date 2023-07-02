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
      octopusdeploy_azure_cloud_service_deployment_target.azure.id,
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