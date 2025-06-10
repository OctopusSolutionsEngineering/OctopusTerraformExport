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

resource "octopusdeploy_project" "project_1" {
  auto_create_release                  = false
  default_guided_failure_mode          = "EnvironmentDefault"
  default_to_skip_if_already_installed = false
  description                          = "Test project"
  discrete_channel_release             = false
  is_disabled                          = false
  is_discrete_channel_release          = false
  is_version_controlled                = true
  lifecycle_id                         = data.octopusdeploy_lifecycles.lifecycle_default_lifecycle.lifecycles[0].id
  name                                 = "Test"
  project_group_id                     = octopusdeploy_project_group.project_group_test.id
  tenanted_deployment_participation    = "Untenanted"
  space_id                             = var.octopus_space_id
  included_library_variable_sets       = [octopusdeploy_library_variable_set.octopus_library_variable_set.id]
  versioning_strategy {
    template = "#{Octopus.Version.LastMajor}.#{Octopus.Version.LastMinor}.#{Octopus.Version.LastPatch}.#{Octopus.Version.NextRevision}"
  }

  connectivity_policy {
    allow_deployments_to_no_targets = false
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "SkipUnavailableMachines"
  }

  git_library_persistence_settings {
    git_credential_id  = "${octopusdeploy_git_credential.gitcredential_matt.id}"
    url                = "https://github.com/mcasperson/octogittest.git"
    base_path          = ".octopus/integrationtest-${timestamp()}"
    default_branch     = "main"
    protected_branches = ["test"]
  }
}

resource "octopusdeploy_variable" "string_variable" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "Test"
  value     = "PlainText"
}

resource "octopusdeploy_variable" "string_variable2" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "Test2"
  value     = "PlainText"
  scope {
    environments = [octopusdeploy_environment.development_environment.id]
  }
}

resource "octopusdeploy_variable" "scoped_var" {
  owner_id = octopusdeploy_project.project_1.id
  type     = "String"
  name     = "tenantscoped"
  value    = "test"
  scope {
    tenant_tags = ["tag1/a"]
  }
}

resource "octopusdeploy_variable" "string_variable3" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "Test3"
  value     = "PlainText"
  scope {
    environments = [octopusdeploy_environment.test_environment.id]
  }
}

resource "octopusdeploy_variable" "string_variable4" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "HelmFeed"
  value     = octopusdeploy_helm_feed.feed_helm.id
}

resource "octopusdeploy_variable" "string_variable5" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "UsernamePassword"
  value     = octopusdeploy_username_password_account.account_gke.id
}

resource "octopusdeploy_variable" "string_variable6" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "WorkerPool"
  value     = octopusdeploy_static_worker_pool.workerpool_docker.id
}

resource "octopusdeploy_variable" "string_variable7" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "Certificate"
  value     = octopusdeploy_certificate.certificate_kind_ca.id
}

resource "octopusdeploy_variable" "string_variable8" {
  owner_id  = octopusdeploy_project.project_1.id
  type      = "String"
  name      = "TestNull"
  value     = null
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
  project_id = "${octopusdeploy_project.project_2.id}"
  depends_on = []
}
resource "octopusdeploy_process_steps_order" "test" {
  process_id = "${octopusdeploy_process.test.id}"
  steps      = ["${octopusdeploy_process_step.hello_world.id}"]
}

resource "octopusdeploy_process_step" "hello_world" {
  name                  = "Hello world (using Bash)"
  type                  = "Octopus.Script"
  process_id            = "${octopusdeploy_process.test.id}"
  channels              = null
  condition             = "Success"
  environments          = null
  excluded_environments = null
  package_requirement   = "LetOctopusDecide"
  packages              = { package1 = { acquisition_location = "ExecutionTarget", feed_id = "${octopusdeploy_docker_container_registry.feed_docker.id}", id = null, package_id = "package1", properties = { Extract = "False", Purpose = "", SelectionMode = "immediate" } } }
  slug                  = "hello-world"
  start_trigger         = "StartAfterPrevious"
  tenant_tags           = null
  execution_properties  = {
    "Octopus.Action.Script.ScriptSource" = "Inline"
    "Octopus.Action.Script.ScriptBody" = "echo 'Hello world, using Bash'\n\n#TODO: Experiment with steps of your own :)\n\necho '[Learn more about the types of steps available in Octopus](https://oc.to/OnboardingAddStepsLearnMore)'"
    "Octopus.Action.Script.Syntax" = "Bash"
    "Octopus.Action.RunOnServer" = "true"
  }
  properties            = {

  }
}
