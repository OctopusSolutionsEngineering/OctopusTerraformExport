data "octopusdeploy_lifecycles" "lifecycle_default_lifecycle" {
  ids          = null
  partial_name = "Default Lifecycle"
  skip         = 0
  take         = 1
}

data "octopusdeploy_project_groups" "project_group" {
  ids          = null
  partial_name = "Test"
  skip         = 0
  take         = 1
}

data "octopusdeploy_kubernetes_cluster_deployment_targets" "test" {
  environments = []
  ids          = []
  partial_name = "Test"
  name         = "Test"
  skip         = 0
  take         = 1
}

data "octopusdeploy_deployment_targets" "ssh" {
  partial_name = "Ssh"
  skip         = 0
  take         = 1
}

data "octopusdeploy_deployment_targets" "polling" {
  partial_name = "Polling"
  skip         = 0
  take         = 1
}

data "octopusdeploy_deployment_targets" "listening" {
  partial_name = "Listening"
  skip         = 0
  take         = 1
}

data "octopusdeploy_deployment_targets" "offline" {
  partial_name = "Offline"
  skip         = 0
  take         = 1
}

# data "octopusdeploy_deployment_targets" "azure" {
#   partial_name = "Azure"
#   skip         = 0
#   take         = 1
# }

data "octopusdeploy_deployment_targets" "service_facbric" {
  partial_name = "ServiceFabric"
  skip         = 0
  take         = 1
}

data "octopusdeploy_deployment_targets" "webapp" {
  partial_name = "WebApp"
  skip         = 0
  take         = 1
}

data "octopusdeploy_deployment_targets" "cloud" {
  partial_name = "Cloud"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_project" "project_1" {
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
      data.octopusdeploy_kubernetes_cluster_deployment_targets.test.kubernetes_cluster_deployment_targets[0].id,
      data.octopusdeploy_deployment_targets.cloud.deployment_targets[0].id,
      data.octopusdeploy_deployment_targets.ssh.deployment_targets[0].id,
      data.octopusdeploy_deployment_targets.polling.deployment_targets[0].id,
      data.octopusdeploy_deployment_targets.listening.deployment_targets[0].id,
      data.octopusdeploy_deployment_targets.offline.deployment_targets[0].id,
      #data.octopusdeploy_deployment_targets.azure.deployment_targets[0].id,
      data.octopusdeploy_deployment_targets.service_facbric.deployment_targets[0].id,
      data.octopusdeploy_deployment_targets.webapp.deployment_targets[0].id,
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
  lifecycle_id                         = data.octopusdeploy_lifecycles.lifecycle_default_lifecycle.lifecycles[0].id
  name                                 = "Test 2"
  project_group_id                     = data.octopusdeploy_project_groups.project_group.project_groups[0].id
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